/*
Copyright 2025 kspec contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kspecv1alpha1 "github.com/cloudcwfranck/kspec/api/v1alpha1"
	"github.com/cloudcwfranck/kspec/pkg/aggregation"
)

var (
	k8sClient  client.Client
	aggregator *aggregation.ReportAggregator
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Initialize Kubernetes client
	if err := initKubernetesClient(); err != nil {
		log.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Setup HTTP handlers
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/api/summary", handleAPISummary)
	http.HandleFunc("/api/clusters", handleAPIClusters)
	http.HandleFunc("/api/failures", handleAPIFailures)
	http.HandleFunc("/health", handleHealth)

	// Start server
	log.Printf("Starting kspec web dashboard on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func initKubernetesClient() error {
	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to local config for development
		config = ctrl.GetConfigOrDie()
	}

	// Create scheme
	scheme := runtime.NewScheme()
	if err := kspecv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add scheme: %w", err)
	}

	// Create client
	k8sClient, err = client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	aggregator = aggregation.NewReportAggregator(k8sClient)
	return nil
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleAPISummary(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	clusterSpec := r.URL.Query().Get("cluster_spec")

	// Get all ClusterSpecs if not specified
	if clusterSpec == "" {
		var clusterSpecs kspecv1alpha1.ClusterSpecificationList
		if err := k8sClient.List(ctx, &clusterSpecs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(clusterSpecs.Items) > 0 {
			clusterSpec = clusterSpecs.Items[0].Name
		}
	}

	if clusterSpec == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "No ClusterSpecifications found",
		})
		return
	}

	summary, err := aggregator.GetFleetSummary(ctx, clusterSpec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func handleAPIClusters(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	clusterSpec := r.URL.Query().Get("cluster_spec")

	if clusterSpec == "" {
		var clusterSpecs kspecv1alpha1.ClusterSpecificationList
		if err := k8sClient.List(ctx, &clusterSpecs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(clusterSpecs.Items) > 0 {
			clusterSpec = clusterSpecs.Items[0].Name
		}
	}

	clusters, err := aggregator.GetClusterCompliance(ctx, clusterSpec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get cluster targets for additional info
	targets, _ := aggregator.GetClusterTargets(ctx, "")

	// Enrich cluster data
	targetMap := make(map[string]*kspecv1alpha1.ClusterTarget)
	for i := range targets {
		targetMap[targets[i].Name] = &targets[i]
	}

	type EnrichedCluster struct {
		aggregation.ClusterCompliance
		Platform  string `json:"platform"`
		Nodes     int32  `json:"nodes"`
		Reachable bool   `json:"reachable"`
		Version   string `json:"version"`
	}

	enriched := make([]EnrichedCluster, len(clusters))
	for i, c := range clusters {
		ec := EnrichedCluster{
			ClusterCompliance: c,
			Platform:          "Unknown",
			Reachable:         true,
		}
		if target, ok := targetMap[c.ClusterName]; ok {
			ec.Platform = target.Status.Platform
			ec.Nodes = target.Status.NodeCount
			ec.Reachable = target.Status.Reachable
			ec.Version = target.Status.Version
		} else if c.IsLocal {
			ec.Platform = "Local"
		}
		enriched[i] = ec
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(enriched)
}

func handleAPIFailures(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	clusterSpec := r.URL.Query().Get("cluster_spec")

	if clusterSpec == "" {
		var clusterSpecs kspecv1alpha1.ClusterSpecificationList
		if err := k8sClient.List(ctx, &clusterSpecs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(clusterSpecs.Items) > 0 {
			clusterSpec = clusterSpecs.Items[0].Name
		}
	}

	failures, err := aggregator.GetFailedChecksByCluster(ctx, clusterSpec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(failures)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

const dashboardHTML = `<!DOCTYPE html>
<html>
<head>
    <title>kspec Compliance Dashboard</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f5f7fa;
            color: #2c3e50;
            padding: 20px;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        h1 { font-size: 2em; margin-bottom: 10px; }
        .subtitle { opacity: 0.9; font-size: 0.9em; }
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .card {
            background: white;
            padding: 25px;
            border-radius: 10px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .card h3 {
            font-size: 0.9em;
            color: #7f8c8d;
            text-transform: uppercase;
            margin-bottom: 10px;
            letter-spacing: 0.5px;
        }
        .card .value {
            font-size: 2.5em;
            font-weight: bold;
            color: #2c3e50;
        }
        .card .subvalue {
            color: #95a5a6;
            font-size: 0.9em;
            margin-top: 5px;
        }
        .compliance-high { color: #27ae60; }
        .compliance-medium { color: #f39c12; }
        .compliance-low { color: #e74c3c; }
        table {
            width: 100%;
            background: white;
            border-radius: 10px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        th, td {
            padding: 15px;
            text-align: left;
            border-bottom: 1px solid #ecf0f1;
        }
        th {
            background: #34495e;
            color: white;
            font-weight: 600;
            text-transform: uppercase;
            font-size: 0.85em;
            letter-spacing: 0.5px;
        }
        tr:last-child td { border-bottom: none; }
        tr:hover { background: #f8f9fa; }
        .status-badge {
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
            display: inline-block;
        }
        .status-healthy { background: #d4edda; color: #155724; }
        .status-warning { background: #fff3cd; color: #856404; }
        .status-error { background: #f8d7da; color: #721c24; }
        .progress-bar {
            width: 100%;
            height: 8px;
            background: #ecf0f1;
            border-radius: 4px;
            overflow: hidden;
            margin-top: 10px;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
            transition: width 0.3s ease;
        }
        .loading {
            text-align: center;
            padding: 40px;
            color: #95a5a6;
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 20px;
            border-radius: 10px;
            margin: 20px 0;
        }
        .refresh-info {
            text-align: center;
            color: #95a5a6;
            font-size: 0.9em;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🛡️ kspec Compliance Dashboard</h1>
            <div class="subtitle">Real-time multi-cluster compliance monitoring</div>
            <div class="subtitle" id="last-update">Loading...</div>
        </header>

        <div class="summary-grid" id="summary">
            <div class="loading">Loading fleet summary...</div>
        </div>

        <div class="card" style="margin-bottom: 30px;">
            <h3>Cluster Status</h3>
            <table id="clusters">
                <thead>
                    <tr>
                        <th>Cluster</th>
                        <th>Compliance</th>
                        <th>Checks</th>
                        <th>Drift</th>
                        <th>Platform</th>
                        <th>Nodes</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    <tr><td colspan="7" class="loading">Loading clusters...</td></tr>
                </tbody>
            </table>
        </div>
    </div>

    <script>
        function fetchData() {
            // Fetch summary
            fetch('/api/summary')
                .then(r => r.json())
                .then(data => {
                    if (data.error) {
                        document.getElementById('summary').innerHTML =
                            '<div class="error">' + data.error + '</div>';
                        return;
                    }
                    updateSummary(data);
                })
                .catch(err => {
                    document.getElementById('summary').innerHTML =
                        '<div class="error">Failed to load summary: ' + err + '</div>';
                });

            // Fetch clusters
            fetch('/api/clusters')
                .then(r => r.json())
                .then(data => updateClusters(data))
                .catch(err => {
                    document.getElementById('clusters').querySelector('tbody').innerHTML =
                        '<tr><td colspan="7" class="error">Failed to load clusters: ' + err + '</td></tr>';
                });

            // Update timestamp
            document.getElementById('last-update').textContent =
                'Last updated: ' + new Date().toLocaleString();
        }

        function updateSummary(data) {
            const compliancePercent = data.TotalChecks > 0
                ? (data.PassedChecks / data.TotalChecks * 100).toFixed(1)
                : 0;

            let complianceClass = 'compliance-low';
            if (compliancePercent >= 95) complianceClass = 'compliance-high';
            else if (compliancePercent >= 80) complianceClass = 'compliance-medium';

            const html = ` +
		`<div class="card">
                    <h3>Overall Compliance</h3>
                    <div class="value ${complianceClass}">${compliancePercent}%</div>
                    <div class="subvalue">${data.PassedChecks}/${data.TotalChecks} checks passed</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${compliancePercent}%"></div>
                    </div>
                </div>
                <div class="card">
                    <h3>Clusters</h3>
                    <div class="value">${data.TotalClusters}</div>
                    <div class="subvalue">
                        <span style="color: #27ae60;">✓ ${data.HealthyClusters} healthy</span>
                        ${data.UnhealthyClusters > 0 ? '<br><span style="color: #e74c3c;">✗ ' + data.UnhealthyClusters + ' unhealthy</span>' : ''}
                    </div>
                </div>
                <div class="card">
                    <h3>Drift Status</h3>
                    <div class="value ${data.ClustersWithDrift > 0 ? 'compliance-medium' : 'compliance-high'}">
                        ${data.ClustersWithDrift > 0 ? '⚡' : '✓'}
                    </div>
                    <div class="subvalue">
                        ${data.ClustersWithDrift > 0
                            ? data.ClustersWithDrift + ' clusters with drift'
                            : 'No drift detected'}
                    </div>
                </div>
                <div class="card">
                    <h3>Failed Checks</h3>
                    <div class="value ${data.FailedChecks > 0 ? 'compliance-low' : 'compliance-high'}">
                        ${data.FailedChecks}
                    </div>
                    <div class="subvalue">Total failures across fleet</div>
                </div>
            ` + "`" + `;
            document.getElementById('summary').innerHTML = html;
        }

        function updateClusters(data) {
            if (!data || data.length === 0) {
                document.getElementById('clusters').querySelector('tbody').innerHTML =
                    '<tr><td colspan="7" style="text-align: center; padding: 40px; color: #95a5a6;">No clusters found</td></tr>';
                return;
            }

            const rows = data.map(c => {
                const compliancePercent = c.ComplianceScore.toFixed(1);
                let complianceClass = 'status-error';
                if (c.ComplianceScore >= 95) complianceClass = 'status-healthy';
                else if (c.ComplianceScore >= 80) complianceClass = 'status-warning';

                const statusClass = c.Reachable ? 'status-healthy' : 'status-error';
                const statusText = c.Reachable ? '✓ Healthy' : '✗ Unreachable';

                return ` + "`" + `<tr>
                    <td><strong>${c.ClusterName}</strong></td>
                    <td><span class="status-badge ${complianceClass}">${compliancePercent}%</span></td>
                    <td>${c.PassedChecks}/${c.TotalChecks}</td>
                    <td>${c.HasDrift ? '⚡ ' + c.DriftEventCount + ' events' : '✓ None'}</td>
                    <td>${c.Platform || 'Unknown'}</td>
                    <td>${c.Nodes || '-'}</td>
                    <td><span class="status-badge ${statusClass}">${statusText}</span></td>
                </tr>` + "`" + `;
            }).join('');

            document.getElementById('clusters').querySelector('tbody').innerHTML = rows;
        }

        // Initial load
        fetchData();

        // Auto-refresh every 30 seconds
        setInterval(fetchData, 30000);
    </script>
</body>
</html>
`
