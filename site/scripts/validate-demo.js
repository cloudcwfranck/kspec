#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Read demo data and schema
const demoDataPath = path.join(__dirname, '../demo/demoSteps.json');
const schemaPath = path.join(__dirname, '../demo/demoStepsSchema.json');

try {
  const demoData = JSON.parse(fs.readFileSync(demoDataPath, 'utf8'));
  const schema = JSON.parse(fs.readFileSync(schemaPath, 'utf8'));

  console.log('Validating demo data...\n');

  let errors = [];

  // Validate version format
  const versionRegex = /^\d+\.\d+\.\d+$/;
  if (!versionRegex.test(demoData.version)) {
    errors.push(`Invalid version format: ${demoData.version} (expected semver)`);
  }

  // Validate tabs
  if (!Array.isArray(demoData.tabs) || demoData.tabs.length === 0) {
    errors.push('Tabs must be a non-empty array');
  } else {
    demoData.tabs.forEach((tab, index) => {
      // Validate tab ID (kebab-case)
      const idRegex = /^[a-z][a-z0-9-]*$/;
      if (!idRegex.test(tab.id)) {
        errors.push(`Tab ${index}: Invalid ID '${tab.id}' (must be kebab-case)`);
      }

      // Validate required fields
      if (!tab.label || tab.label.length === 0) {
        errors.push(`Tab ${index}: Missing or empty label`);
      }
      if (!tab.description || tab.description.length === 0) {
        errors.push(`Tab ${index}: Missing or empty description`);
      }

      // Validate docs link (allow internal /docs/ or external GitHub URLs)
      const docsLinkRegex = /^(\/docs\/.*|https:\/\/github\.com\/.*)$/;
      if (!tab.docsLink || !docsLinkRegex.test(tab.docsLink)) {
        errors.push(`Tab ${index}: Invalid docsLink '${tab.docsLink}' (must start with /docs/ or be a GitHub URL)`);
      }

      // Validate steps
      if (!Array.isArray(tab.steps) || tab.steps.length === 0) {
        errors.push(`Tab ${index}: Steps must be a non-empty array`);
      } else {
        tab.steps.forEach((step, stepIndex) => {
          if (!step.command || step.command.length === 0) {
            errors.push(`Tab ${index}, Step ${stepIndex}: Missing or empty command`);
          }
          if (typeof step.output !== 'string') {
            errors.push(`Tab ${index}, Step ${stepIndex}: Output must be a string`);
          }
        });
      }
    });
  }

  if (errors.length > 0) {
    console.error('Validation failed with the following errors:\n');
    errors.forEach((error, index) => {
      console.error(`  ${index + 1}. ${error}`);
    });
    console.error('\n');
    process.exit(1);
  } else {
    console.log('✓ Demo data is valid!');
    console.log(`  Version: ${demoData.version}`);
    console.log(`  Tabs: ${demoData.tabs.length}`);
    console.log(`  Total steps: ${demoData.tabs.reduce((sum, tab) => sum + tab.steps.length, 0)}`);
    console.log('\n');
  }
} catch (error) {
  console.error('Error validating demo data:');
  console.error(error.message);
  process.exit(1);
}
