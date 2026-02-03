# Troubleshooting Guide - Common Issues & Solutions

## Table of Contents

**By Category:**
1. [Alignment Problems](#alignment-problems) - Text/checkbox alignment, overflow, flex centering (3 issues)
2. [Color & Styling Issues](#color--styling-issues) - CSS variables, borders, shadows (3 issues)
3. [Spacing Mismatches](#spacing-mismatches) - Gap approximations, padding arrays (2 issues)
4. [Font Rendering](#font-rendering) - Font weight, line height (2 issues)
5. [Pencil MCP Tool Errors](#pencil-mcp-tool-errors) - batch_get, get_variables, node IDs (3 issues)
6. [TypeScript Errors](#typescript-errors) - Style types, imports (2 issues)
7. [Layout Issues](#layout-issues) - Flex containers, absolute positioning (2 issues)

**Quick Diagnostic:** See bottom of document for diagnostic checklist.

---

## Alignment Problems

### Problem 1: Text Not Aligned with Checkbox/Radio

**Symptom:**
```typescript
// Text baseline doesn't match checkbox center
<Checkbox className="w-[18px] h-[18px]" />
<span>Label text</span>  // Appears lower than checkbox
```

**Root Cause:**
Text elements have variable height based on font metrics. Checkbox is fixed 18px.

**Solution:**
```typescript
// Fix label to same height as checkbox
<Checkbox className="w-[18px] h-[18px]" />
<Label className="h-[18px] flex items-center">
  Label text
</Label>
```

**Prevention:**
Always check Pencil frame's `alignItems` property. If it's `center` with fixed-height children, apply fixed height to text elements.

---

### Problem 2: Nested Elements Overflow Parent

**Symptom:**
```typescript
// Child elements extend beyond parent bounds
<div style={{ width: '400px' }}>
  <div style={{ width: '450px' }}>  // Overflows!
    Content
  </div>
</div>
```

**Root Cause:**
Pencil's `fill_container` not converted correctly, or explicit width set on child.

**Solution:**
```typescript
// Use w-full or width: '100%' for children
<div style={{ width: '400px' }}>
  <div className="w-full">  // Respects parent
    Content
  </div>
</div>
```

**Check in Pencil:**
Look for `width: "fill_container"` in child nodes.

---

### Problem 3: Flex Items Not Centering

**Symptom:**
```typescript
// Items appear at top despite alignItems: center
<div style={{ display: 'flex', alignItems: 'center' }}>
  <Icon />
  <Text />  // Still at top!
</div>
```

**Root Cause:**
Missing `height` on flex container, so `alignItems` has no effect.

**Solution:**
```typescript
// Add explicit height or min-height
<div style={{
  display: 'flex',
  alignItems: 'center',
  height: '36px'  // Now center works!
}}>
  <Icon />
  <Text />
</div>
```

---

## Color & Styling Issues

### Problem 4: Colors Don't Match Pencil

**Symptom:**
```typescript
className="bg-[bg-primary]"  // Renders as invalid color
```

**Root Cause:**
Tailwind doesn't support CSS variable references in arbitrary values.

**Solution:**
```typescript
// Step 1: Get variables
const vars = mcp__pencil__get_variables({ filePath: "design.pen" })

// Step 2: Map to hex
// vars = { "bg-primary": "#1e1e1e" }

// Step 3: Use hex in code
className="bg-[#1e1e1e]"
```

**Automation:**
Create a helper function during conversion:
```typescript
const colorMap = {
  'bg-primary': '#1e1e1e',
  'border-subtle': '#3c3c3c',
  // ... from get_variables()
};

const getColor = (varName: string) => colorMap[varName] || varName;
```

---

### Problem 5: Border Rendering Incorrectly

**Symptom:**
```typescript
// Border too thick or wrong color
style={{ border: '1px solid #000000' }}  // But looks different
```

**Root Cause:**
Pencil's `stroke` and `strokeThickness` not mapped correctly.

**Solution:**
```typescript
// Check Pencil properties:
// stroke: "#3c3c3c"
// strokeThickness: 1
// strokePosition: "inside" | "center" | "outside"

// For inside borders (most common):
style={{ border: '1px solid #3c3c3c' }}

// For outside borders:
style={{
  boxShadow: '0 0 0 1px #3c3c3c',  // Simulates outside border
  border: 'none'
}}
```

---

### Problem 6: Shadow Not Appearing

**Symptom:**
```typescript
// Pencil has shadow, but it doesn't show in code
style={{ boxShadow: 'none' }}
```

**Root Cause:**
Pencil `shadow` object not converted.

**Solution:**
```typescript
// Pencil shadow object:
// {
//   "x": 0,
//   "y": 2,
//   "blur": 8,
//   "spread": 0,
//   "color": "#00000026"  // rgba(0,0,0,0.15)
// }

// Convert to CSS:
style={{
  boxShadow: '0 2px 8px 0 rgba(0, 0, 0, 0.15)'
}}

// Format: offsetX offsetY blur spread color
```

---

## Spacing Mismatches

### Problem 7: Gap Appears Wrong

**Symptom:**
```typescript
className="space-y-2"  // 8px, but Pencil shows 6px
```

**Root Cause:**
Tailwind units are rem-based (space-2 = 0.5rem = 8px).

**Solution:**
```typescript
// Never use Tailwind spacing for exact matches
// Use inline styles with exact pixels
style={{ gap: '6px' }}  // Pencil gap: 6
```

**Tailwind → Pixel Conversion:**
- `space-1` = 4px
- `space-2` = 8px
- `space-3` = 12px
- `space-4` = 16px

**Rule:** If Pencil gap is not a multiple of 4, ALWAYS use inline styles.

---

### Problem 8: Padding Array Confusion

**Symptom:**
```typescript
// Pencil padding: [10, 12]
style={{ padding: '10px 12px' }}  // Works

// But what about [2, 0, 0, 0]?
style={{ padding: '2px 0 0 0' }}  // Correct?
```

**Root Cause:**
Confusion about CSS shorthand vs. explicit properties.

**Solution:**
```typescript
// Single value
[10] → padding: '10px'

// Two values (vertical, horizontal)
[10, 12] → padding: '10px 12px'

// Four values (top, right, bottom, left)
[10, 12, 14, 16] → padding: '10px 12px 14px 16px'

// Special case: Only top padding
[2, 0, 0, 0] → paddingTop: '2px'  // More explicit
```

---

## Font Rendering

### Problem 9: Font Weight Looks Different

**Symptom:**
```typescript
style={{ fontWeight: '500' }}  // Appears as regular (400)
```

**Root Cause:**
Font weight passed as string instead of number.

**Solution:**
```typescript
// ❌ Wrong
style={{ fontWeight: '500' }}

// ✅ Correct
style={{ fontWeight: 500 }}

// Pencil uses strings, convert to numbers
```

---

### Problem 10: Line Height Causes Layout Issues

**Symptom:**
```typescript
// Text takes more vertical space than expected
style={{ fontSize: '14px', lineHeight: 1.5 }}  // Too much space
```

**Root Cause:**
Line height multiplier creates extra space above/below text.

**Solution:**
```typescript
// For inline elements that need precise height:
<span style={{
  fontSize: '14px',
  lineHeight: 1,  // Tight line height
  display: 'block'
}}>
  Text
</span>

// Or use fixed container height:
<div className="h-[18px] flex items-center">
  <span style={{ fontSize: '14px', lineHeight: 1.5 }}>
    Text
  </span>
</div>
```

---

## Pencil MCP Tool Errors

### Problem 11: batch_get Returns Too Much Data

**Symptom:**
```
Error: Response too large (exceeded token limit)
```

**Root Cause:**
`readDepth` too high on large component tree.

**Solution:**
```typescript
// Start with shallow depth
mcp__pencil__batch_get({
  filePath: "design.pen",
  nodeIds: ["root"],
  readDepth: 2  // Low depth first
})

// Then drill down into specific nodes
mcp__pencil__batch_get({
  filePath: "design.pen",
  nodeIds: ["specificChild"],
  readDepth: 5  // Deep dive on small subtree
})
```

---

### Problem 12: get_variables Returns Empty

**Symptom:**
```typescript
const vars = mcp__pencil__get_variables({ filePath: "design.pen" })
// Returns: {}
```

**Root Cause:**
Design file has no variables defined, or uses raw colors.

**Solution:**
```typescript
// Check if variables exist
if (Object.keys(vars).length === 0) {
  // No variables, all colors are direct hex values
  // Extract colors from batch_get result instead
}

// Alternative: Look for fill/stroke properties in nodes
// fill: "#1e1e1e"  // Direct hex
// fill: "$bg-primary"  // Variable reference (starts with $)
```

---

### Problem 13: Node ID Not Found

**Symptom:**
```
Error: Node with ID "xyz" not found
```

**Root Cause:**
Using stale node ID, or incorrect path for nested component nodes.

**Solution:**
```typescript
// For component instances, use path notation
// Instance ID: "myButton"
// Child ID: "label"

// ❌ Wrong
nodeIds: ["label"]  // Won't find shadow node

// ✅ Correct
nodeIds: ["myButton/label"]  // Path notation
```

---

## TypeScript Errors

### Problem 14: Type Error on Style Properties

**Symptom:**
```typescript
style={{ gap: '6px' }}  // Error: Type 'string' not assignable
```

**Root Cause:**
React's CSSProperties expects number for certain properties in some cases.

**Solution:**
```typescript
// Use correct type based on property

// Gap, padding, margin: string with units
style={{ gap: '6px', padding: '10px' }}

// Dimensions: string with units or number (px assumed)
style={{ width: '200px', height: 36 }}

// Numbers without units: use number type
style={{ opacity: 0.5, zIndex: 10, fontWeight: 500 }}
```

---

### Problem 15: Missing Import Errors

**Symptom:**
```
Error: 'React' must be in scope when using JSX
```

**Solution:**
```typescript
// Always include at top of file
import React from 'react';

// For newer React (17+), optional but recommended
import { FC } from 'react';
```

---

## Layout Issues

### Problem 16: Flex Container Not Working

**Symptom:**
```typescript
<div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
  <div>Item 1</div>
  <div>Item 2</div>
</div>
// Items appear side-by-side instead of stacked
```

**Root Cause:**
CSS not applied correctly, or browser compatibility issue.

**Solution:**
```typescript
// Ensure all flex properties are set
<div style={{
  display: 'flex',
  flexDirection: 'column',  // Explicit column
  gap: '10px'
}}>
  <div>Item 1</div>
  <div>Item 2</div>
</div>

// Or use Tailwind utilities
<div className="flex flex-col gap-[10px]">
  <div>Item 1</div>
  <div>Item 2</div>
</div>
```

---

### Problem 17: Absolute Positioning Breaks Layout

**Symptom:**
```typescript
// Element appears outside parent bounds
<div style={{ position: 'relative' }}>
  <div style={{ position: 'absolute', top: 0, left: 0 }}>
    Content
  </div>
</div>
```

**Root Cause:**
Pencil uses absolute positioning, but parent needs explicit dimensions.

**Solution:**
```typescript
// Parent must have dimensions for absolute child
<div style={{
  position: 'relative',
  width: '400px',    // Explicit width
  height: '200px'    // Explicit height
}}>
  <div style={{ position: 'absolute', top: 0, left: 0 }}>
    Content
  </div>
</div>
```

---

## Quick Diagnostic Checklist

When something doesn't look right:

1. **Colors wrong?** → Check if CSS variables converted to hex
2. **Spacing off?** → Verify exact pixels, not Tailwind units
3. **Text misaligned?** → Add fixed height to text container
4. **Font looks different?** → Check fontWeight is number, not string
5. **Layout broken?** → Verify flex properties and parent dimensions
6. **Border wrong?** → Check strokeThickness and strokePosition
7. **Too much data?** → Lower readDepth in batch_get
8. **Node not found?** → Use path notation for component children

---

## Getting Help

If issues persist after troubleshooting:

1. **Compare screenshots**: Pencil vs. actual render side-by-side
2. **Use DevTools**: Inspect computed styles
3. **Check Pencil data**: Re-run batch_get with higher readDepth
4. **Verify variables**: Ensure get_variables() called and colors mapped
5. **Validate structure**: Confirm Pencil hierarchy matches React structure

**Last Resort:**
Re-analyze the Pencil design from scratch with fresh tool calls.
