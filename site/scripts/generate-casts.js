#!/usr/bin/env node
/**
 * Generate Asciinema cast files from demoSteps.json
 */

const fs = require('fs');
const path = require('path');

const demoData = JSON.parse(fs.readFileSync(path.join(__dirname, '../demo/demoSteps.json'), 'utf8'));

const COLORS = {
  green: '\u001b[32m',
  red: '\u001b[31m',
  yellow: '\u001b[33m',
  reset: '\u001b[0m',
};

function escapeAnsi(text) {
  return text
    .replace(/⠿/g, '\u001b[33m⠿\u001b[0m')
    .replace(/✓/g, `${COLORS.green}✓${COLORS.reset}`)
    .replace(/✗/g, `${COLORS.red}✗${COLORS.reset}`)
    .replace(/⚠/g, `${COLORS.yellow}⚠${COLORS.reset}`)
    .replace(/PASS/g, `${COLORS.green}PASS${COLORS.reset}`)
    .replace(/FAIL/g, `${COLORS.red}FAIL${COLORS.reset}`);
}

function typeCommand(command, startTime) {
  const events = [];
  let time = startTime;

  for (const char of command) {
    events.push([time, "o", char]);
    time += 0.1;
  }

  events.push([time, "o", "\r\n"]);
  return { events, endTime: time + 0.1 };
}

function printOutput(output, startTime) {
  const events = [];
  let time = startTime;

  const lines = output.split('\n');
  for (const line of lines) {
    const coloredLine = escapeAnsi(line);
    events.push([time, "o", coloredLine + "\r\n"]);
    time += 0.1;
  }

  return { events, endTime: time };
}

function generateCast(tab) {
  const header = {
    version: 2,
    width: 120,
    height: 30,
    timestamp: Math.floor(Date.now() / 1000),
    env: {
      SHELL: "/bin/bash",
      TERM: "xterm-256color"
    },
    title: `kspec ${tab.id} demo`,
    idle_time_limit: 2.0
  };

  let events = [];
  let time = 0.5;

  // Initial prompt
  const prompt = `${COLORS.green}(kind-kspec)${COLORS.reset} franck@csengineering$ `;
  events.push([time, "o", prompt]);
  time += 1.0;

  // Process each step
  for (const step of tab.steps) {
    // Type command
    const cmdResult = typeCommand(step.command, time);
    events.push(...cmdResult.events);
    time = cmdResult.endTime + 0.2;

    // Show output
    const outResult = printOutput(step.output, time);
    events.push(...outResult.events);
    time = outResult.endTime + 0.5;

    // Show prompt again
    events.push([time, "o", prompt]);
    time += 1.0;
  }

  // Create cast content
  const lines = [JSON.stringify(header)];
  for (const event of events) {
    lines.push(JSON.stringify(event));
  }

  return lines.join('\n');
}

// Generate all casts
const outputDir = path.join(__dirname, '../public/demos/asciinema');
fs.mkdirSync(outputDir, { recursive: true });

for (const tab of demoData.tabs) {
  const castContent = generateCast(tab);
  const outputPath = path.join(outputDir, `${tab.id}.cast`);
  fs.writeFileSync(outputPath, castContent);
  console.log(`✓ Generated ${tab.id}.cast`);
}

console.log(`\nGenerated ${demoData.tabs.length} cast files in ${outputDir}`);
