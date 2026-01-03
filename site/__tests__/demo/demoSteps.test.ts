import { describe, it, expect } from 'vitest';
import demoData from '@/demo/demoSteps.json';
import schema from '@/demo/demoStepsSchema.json';

describe('Demo Steps Data', () => {
  describe('Schema Compliance', () => {
    it('has valid version format', () => {
      expect(demoData.version).toMatch(/^\d+\.\d+\.\d+$/);
    });

    it('has tabs array', () => {
      expect(Array.isArray(demoData.tabs)).toBe(true);
      expect(demoData.tabs.length).toBeGreaterThan(0);
    });

    it('all tabs have required fields', () => {
      demoData.tabs.forEach((tab, index) => {
        expect(tab.id, `Tab ${index} missing id`).toBeDefined();
        expect(tab.label, `Tab ${index} missing label`).toBeDefined();
        expect(tab.description, `Tab ${index} missing description`).toBeDefined();
        expect(tab.steps, `Tab ${index} missing steps`).toBeDefined();
        expect(tab.docsLink, `Tab ${index} missing docsLink`).toBeDefined();
      });
    });

    it('all tab IDs are kebab-case', () => {
      const kebabCaseRegex = /^[a-z][a-z0-9-]*$/;

      demoData.tabs.forEach((tab, index) => {
        expect(tab.id, `Tab ${index} ID '${tab.id}' is not kebab-case`).toMatch(kebabCaseRegex);
      });
    });

    it('all tab IDs are unique', () => {
      const ids = demoData.tabs.map(tab => tab.id);
      const uniqueIds = new Set(ids);

      expect(uniqueIds.size).toBe(ids.length);
    });

    it('all tabs have non-empty labels', () => {
      demoData.tabs.forEach((tab, index) => {
        expect(tab.label.length, `Tab ${index} has empty label`).toBeGreaterThan(0);
      });
    });

    it('all tabs have non-empty descriptions', () => {
      demoData.tabs.forEach((tab, index) => {
        expect(tab.description.length, `Tab ${index} has empty description`).toBeGreaterThan(0);
      });
    });

    it('all docsLinks are valid paths (internal /docs/ or external GitHub)', () => {
      const validDocsLinkRegex = /^(\/docs\/.*|https:\/\/github\.com\/.*)$/;

      demoData.tabs.forEach((tab, index) => {
        expect(tab.docsLink, `Tab ${index} docsLink '${tab.docsLink}' is not a valid docs path`).toMatch(validDocsLinkRegex);
      });
    });

    it('all tabs have at least one step', () => {
      demoData.tabs.forEach((tab, index) => {
        expect(Array.isArray(tab.steps), `Tab ${index} steps is not an array`).toBe(true);
        expect(tab.steps.length, `Tab ${index} has no steps`).toBeGreaterThan(0);
      });
    });

    it('all steps have required fields', () => {
      demoData.tabs.forEach((tab, tabIndex) => {
        tab.steps.forEach((step, stepIndex) => {
          expect(step.command, `Tab ${tabIndex}, Step ${stepIndex} missing command`).toBeDefined();
          expect(step.output, `Tab ${tabIndex}, Step ${stepIndex} missing output`).toBeDefined();
        });
      });
    });

    it('all steps have non-empty commands', () => {
      demoData.tabs.forEach((tab, tabIndex) => {
        tab.steps.forEach((step, stepIndex) => {
          expect(step.command.length, `Tab ${tabIndex}, Step ${stepIndex} has empty command`).toBeGreaterThan(0);
        });
      });
    });

    it('all outputs are strings', () => {
      demoData.tabs.forEach((tab, tabIndex) => {
        tab.steps.forEach((step, stepIndex) => {
          expect(typeof step.output, `Tab ${tabIndex}, Step ${stepIndex} output is not a string`).toBe('string');
        });
      });
    });
  });

  describe('Content Quality', () => {
    it('has expected number of tabs', () => {
      // We expect 5 tabs: scan, enforce, drift, reports, metrics
      expect(demoData.tabs.length).toBe(5);
    });

    it('has scan capability', () => {
      const scanTab = demoData.tabs.find(tab => tab.id === 'scan');
      expect(scanTab).toBeDefined();
      expect(scanTab?.label).toBe('Scan');
    });

    it('has enforce capability', () => {
      const enforceTab = demoData.tabs.find(tab => tab.id === 'enforce');
      expect(enforceTab).toBeDefined();
      expect(enforceTab?.label).toBe('Enforce');
    });

    it('has drift detection capability', () => {
      const driftTab = demoData.tabs.find(tab => tab.id === 'drift');
      expect(driftTab).toBeDefined();
      expect(driftTab?.label).toBe('Drift Detection');
    });

    it('has reports capability', () => {
      const reportsTab = demoData.tabs.find(tab => tab.id === 'reports');
      expect(reportsTab).toBeDefined();
      expect(reportsTab?.label).toBe('Reports');
    });

    it('has metrics capability', () => {
      const metricsTab = demoData.tabs.find(tab => tab.id === 'metrics');
      expect(metricsTab).toBeDefined();
      expect(metricsTab?.label).toBe('Metrics');
    });

    it('all commands start with valid patterns', () => {
      const validPatterns = [
        /^kspec\s/,           // kspec commands
        /^kubectl\s/,         // kubectl commands
        /^curl\s/,            // curl commands
        /^echo\s/,            // echo commands
      ];

      demoData.tabs.forEach((tab, tabIndex) => {
        tab.steps.forEach((step, stepIndex) => {
          const isValid = validPatterns.some(pattern => pattern.test(step.command));
          expect(isValid, `Tab ${tabIndex}, Step ${stepIndex}: Invalid command pattern '${step.command}'`).toBe(true);
        });
      });
    });

    it('outputs are reasonably sized', () => {
      const MAX_OUTPUT_LENGTH = 5000; // characters

      demoData.tabs.forEach((tab, tabIndex) => {
        tab.steps.forEach((step, stepIndex) => {
          expect(
            step.output.length,
            `Tab ${tabIndex}, Step ${stepIndex}: Output too long (${step.output.length} chars)`
          ).toBeLessThanOrEqual(MAX_OUTPUT_LENGTH);
        });
      });
    });

    it('no step has empty output if command is not echo', () => {
      demoData.tabs.forEach((tab, tabIndex) => {
        tab.steps.forEach((step, stepIndex) => {
          if (!step.command.startsWith('echo')) {
            expect(
              step.output.length,
              `Tab ${tabIndex}, Step ${stepIndex}: Non-echo command has empty output`
            ).toBeGreaterThan(0);
          }
        });
      });
    });
  });

  describe('Schema Structure', () => {
    it('schema has correct metadata', () => {
      expect(schema.$schema).toBe('http://json-schema.org/draft-07/schema#');
      expect(schema.title).toBeDefined();
      expect(schema.description).toBeDefined();
    });

    it('schema requires version and tabs', () => {
      expect(schema.required).toContain('version');
      expect(schema.required).toContain('tabs');
    });

    it('schema validates version format', () => {
      expect(schema.properties.version.pattern).toBeDefined();
    });

    it('schema validates tab structure', () => {
      expect(schema.properties.tabs.type).toBe('array');
      expect(schema.properties.tabs.minItems).toBe(1);
    });
  });
});
