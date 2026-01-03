# kspec Interactive Demo

## Overview

This directory contains the interactive terminal demo displayed on the kspec homepage. The demo provides a deterministic, curated experience showing kspec's core capabilities without requiring a live cluster.

## Architecture

### Components

1. **`DemoTerminal.tsx`** - React component that renders the interactive terminal
2. **`demoSteps.json`** - Curated snapshot data containing commands and outputs
3. **`demoStepsSchema.json`** - JSON schema for validating demo data structure

### Design Principles

- **Deterministic**: All output is pre-recorded; no live cluster required
- **Accessible**: Keyboard navigation, screen reader support, reduced motion support
- **Mobile-friendly**: Responsive layout with touch controls
- **Isolated**: No dependencies on Go code or operator functionality
- **Maintainable**: Curated data is easy to update as features evolve

## Demo Data Contract

### File Structure

```json
{
  "version": "1.0.0",
  "tabs": [
    {
      "id": "unique-tab-id",
      "label": "Tab Label",
      "description": "Brief description of the capability",
      "steps": [
        {
          "command": "kspec command --flag",
          "output": "Command output here..."
        }
      ],
      "docsLink": "/docs/section"
    }
  ]
}
```

### Schema Validation

The `demoSteps.json` file is validated against `demoStepsSchema.json`. Key constraints:

- **Version**: Must be semver format (e.g., "1.0.0")
- **Tab IDs**: Must be kebab-case (e.g., "drift-detection")
- **Docs Links**: Must start with `/docs/`
- **Steps**: At least one step per tab
- **Commands**: Non-empty strings
- **Output**: Strings (can be empty for commands with no output)

### Tab Requirements

Each tab represents a kspec capability:

1. **Scan** - Show cluster scanning and compliance checks
2. **Enforce** - Show policy creation and enforcement
3. **Drift Detection** - Show drift detection and reconciliation
4. **Reports** - Show report export in various formats
5. **Metrics** - Show Prometheus metrics and monitoring

## Component API

### DemoTerminal Component

```tsx
import DemoTerminal from '@/demo/DemoTerminal';

// Usage in a page:
<DemoTerminal />
```

**Props**: None (uses internal state and data from `demoSteps.json`)

### Features

1. **Tab Navigation**
   - Click tabs to switch between capabilities
   - Keyboard: Number keys (1-5) to jump to tabs

2. **Playback Controls**
   - Play/Pause: Space bar or button
   - Next Step: → arrow key or button
   - Previous Step: ← arrow key or button
   - Reset: R key or button

3. **Copy Command**
   - Button to copy current command to clipboard
   - Visual feedback on successful copy

4. **Accessibility**
   - ARIA labels for screen readers
   - Keyboard navigation support
   - Reduced motion support (instant text display)
   - High contrast terminal colors
   - Focus indicators

5. **Responsive Design**
   - Mobile: Touch controls, vertical layout
   - Tablet: Mixed touch/keyboard
   - Desktop: Full keyboard shortcuts

## Updating Demo Data

### When to Update

- New kspec features are released
- Command output format changes
- Compliance standards are updated
- Bug fixes in displayed data

### How to Update

1. Edit `site/demo/demoSteps.json`
2. Validate against schema:
   ```bash
   npm run validate:demo
   ```
3. Test in browser:
   ```bash
   npm run dev
   # Navigate to homepage and test demo
   ```
4. Commit changes with descriptive message

### Best Practices

- **Keep outputs concise**: Trim verbose output to essential information
- **Use realistic data**: Commands and outputs should reflect actual usage
- **Highlight key features**: Focus on what makes kspec unique
- **Maintain consistency**: Use same cluster names, namespaces across tabs
- **Test accessibility**: Verify screen reader announcements, keyboard navigation

## Testing

### Unit Tests

Tests are located in `__tests__/demo/` directory:

- `DemoTerminal.test.tsx` - Component rendering and interaction
- `demoSteps.test.ts` - JSON schema validation

### Running Tests

```bash
# Run all tests
npm test

# Run demo tests only
npm test -- demo

# Run with coverage
npm test -- --coverage
```

### Test Coverage Requirements

- Component rendering: All tabs must render
- Keyboard navigation: All shortcuts must work
- Clipboard functionality: Copy command must succeed
- Tab switching: State must reset correctly
- Schema validation: Demo data must pass validation

## Deployment

### Vercel Configuration

The demo is deployed as part of the Next.js site:

- **Root directory**: `/site`
- **Build command**: `npm run build`
- **Output directory**: `.next`

### Build Validation

Before deployment, ensure:

1. No TypeScript errors:
   ```bash
   npm run type-check
   ```

2. Demo data is valid:
   ```bash
   npm run validate:demo
   ```

3. All tests pass:
   ```bash
   npm test
   ```

4. Build succeeds:
   ```bash
   npm run build
   ```

## Troubleshooting

### Demo not loading

- Check browser console for errors
- Verify `demoSteps.json` is valid JSON
- Ensure file is in correct directory: `site/demo/demoSteps.json`

### Schema validation failing

- Run: `npm run validate:demo`
- Check error message for specific validation failure
- Refer to `demoStepsSchema.json` for constraints

### TypeScript errors

- Ensure `demoSteps.json` matches the TypeScript interfaces in `DemoTerminal.tsx`
- Check that all required fields are present in each tab/step

### Playback issues

- Clear browser cache and reload
- Check for JavaScript errors in console
- Verify `useEffect` hooks are not conflicting

## Future Enhancements

Potential improvements for future releases:

- [ ] Add sound effects (with mute option)
- [ ] Export demo as GIF/video
- [ ] Allow users to customize playback speed
- [ ] Add code syntax highlighting in terminal
- [ ] Expand to show more advanced workflows
- [ ] Add "Try in browser" button (with web-based playground)

## Maintenance

**Owner**: kspec documentation team
**Review frequency**: Every minor release
**Last updated**: 2026-01-03
**Demo data version**: 1.0.0
