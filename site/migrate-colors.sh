#!/bin/bash
# Linear.app Color Migration Script
# Updates all color references to match official Linear design system

echo "🎨 Migrating to Linear.app design system..."

# Find all TSX files in app and components
FILES=$(find app components -name '*.tsx' -type f)

for file in $FILES; do
  echo "Processing: $file"

  # Background colors
  sed -i 's/bg-\[#0a0a0a\]/bg-linear-bg/g' "$file"
  sed -i 's/bg-\[#1a1a1a\]/bg-linear-surface/g' "$file"
  sed -i 's/bg-\[#2a2a2a\]/bg-linear-border/g' "$file"

  # Border colors
  sed -i 's/border-\[#2a2a2a\]/border-linear-border/g' "$file"
  sed -i 's/border-\[#3a3a3a\]/border-linear-border/g' "$file"
  sed -i 's/hover:border-\[#3a3a3a\]/hover:border-linear-text-muted/g' "$file"

  # Text colors
  sed -i 's/text-white/text-linear-text/g' "$file"
  sed -i 's/text-\[#a0a0a0\]/text-linear-text-secondary/g' "$file"
  sed -i 's/text-\[#707070\]/text-linear-text-muted/g' "$file"
  sed -i 's/text-\[#e0e0e0\]/text-linear-text/g' "$file"

  # Accent colors (purple to Linear blue-purple)
  sed -i 's/text-primary-500/text-accent/g' "$file"
  sed -i 's/text-primary-600/text-accent/g' "$file"
  sed -i 's/bg-primary-500/bg-accent/g' "$file"
  sed -i 's/bg-primary-600/bg-accent/g' "$file"
  sed -i 's/hover:text-primary-400/hover:text-accent-hover/g' "$file"
  sed -i 's/hover:bg-primary-500/hover:bg-accent-hover/g' "$file"

  # Hover states
  sed -i 's/hover:text-white/hover:text-linear-text/g' "$file"
  sed -i 's/hover:bg-\[#2a2a2a\]/hover:bg-linear-surface/g' "$file"
  sed -i 's/hover:bg-\[#1a1a1a\]/hover:bg-linear-surface/g' "$file"

  # Specific color combinations
  sed -i 's/bg-primary-600\/10/bg-accent\/10/g' "$file"
  sed -i 's/bg-primary-600\/20/bg-accent\/20/g' "$file"
  sed -i 's/border-primary-600\/20/border-accent\/20/g' "$file"
  sed -i 's/border-primary-600\/30/border-accent\/30/g' "$file"

  # Gradient backgrounds
  sed -i 's/from-primary-600\/10/from-accent\/10/g' "$file"
  sed -i 's/from-primary-600\/20/from-accent\/20/g' "$file"
  sed -i 's/bg-primary-500\/10/bg-accent\/10/g' "$file"

  # Border radius (Linear uses 10-12px)
  sed -i 's/rounded-lg/rounded-xl/g' "$file"
done

echo "✅ Color migration complete!"
echo "📝 Updated $(echo "$FILES" | wc -l) files"
