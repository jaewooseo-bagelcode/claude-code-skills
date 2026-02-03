---
name: pencil-to-code
description: Expert skill for converting Pencil designs to React/TypeScript code with pixel-perfect accuracy. Handles CSS variable conversion, precise pixel spacing, alignment with fixed heights, and font specification matching. Use when converting Pencil designs to React/TypeScript, implementing UI components from Pencil specs, or fixing design-code mismatches.
---

# Pencil to Code Conversion Skill

Convert Pencil designs to React/TypeScript with pixel-perfect accuracy.

## Reference Materials

This skill includes three references for different needs:

- **references/quick-reference.md** - Complete Pencil→CSS property mapping table, tool call templates, and verification checklist. Use for fast lookups during conversion.
- **references/complete-example.md** - Full multi-choice component with all techniques applied. Use to understand complete workflow or learn conversion patterns.
- **references/troubleshooting.md** - 17 common problems organized by category with solutions. Use when debugging alignment, colors, spacing, or tool errors.

**Read references as needed**—they're designed for progressive disclosure.

## Core Conversion Workflow

### Phase 1: Analyze Pencil Design

```typescript
// 1. Get node structure (readDepth: 4-5 for detailed styles)
mcp__pencil__batch_get({
  filePath: "design.pen",
  nodeIds: ["targetNodeId"],
  readDepth: 4
})

// 2. Get color variables (for hex conversion)
mcp__pencil__get_variables({
  filePath: "design.pen"
})

// 3. Get visual reference (for verification)
mcp__pencil__get_screenshot({
  filePath: "design.pen",
  nodeId: "targetNodeId"
})
```

**ReadDepth Strategy:**
- Structure overview: 2-3
- Detailed styling: 4-5
- Performance-critical: 1-2

### Phase 2: Map Structure

Convert Pencil nodes to React/TypeScript following these patterns:

**Layout:**
- `frame (layout: vertical)` → `<div style={{ display: 'flex', flexDirection: 'column' }}>`
- `gap: 10` → `style={{ gap: '10px' }}`
- `padding: [10, 12]` → `style={{ padding: '10px 12px' }}`

**Dimensions:**
- `width: 200` → `style={{ width: '200px' }}`
- `width: "fill_container"` → `className="w-full"`

**Colors:**
- `fill: "#FF0000"` → `style={{ backgroundColor: '#FF0000' }}`
- `fill: "$variable"` → Convert to hex using variables from Phase 1

**Typography:**
- `fontFamily: "Inter"` → `style={{ fontFamily: 'Inter' }}`
- `fontSize: 14` → `style={{ fontSize: '14px' }}`
- `fontWeight: "500"` → `style={{ fontWeight: 500 }}` (number, not string)

**See references/quick-reference.md for complete mapping table.**

### Phase 3: Apply Critical Rules

**Rule 1: CSS Variables → Hex**
```typescript
// ❌ Tailwind doesn't support variable references
className="bg-[bg-primary]"

// ✅ Convert to hex from get_variables()
className="bg-[#1e1e1e]"
```

**Rule 2: Exact Pixels**
```typescript
// ❌ Tailwind approximation creates misalignment
className="space-y-2"  // 8px when Pencil is 6px

// ✅ Exact pixel value
style={{ gap: '6px' }}
```

**Rule 3: Fixed Heights for Alignment**
```typescript
// Pencil: frame (alignItems: center, height: 18)
//   ├─ Radio (18x18)
//   └─ Text (fontSize: 14)

// ✅ Fix text height to match radio
<Radio className="w-[18px] h-[18px]" />
<Label className="h-[18px] flex items-center">Text</Label>
```

**Rule 4: Font Specs 100% Match**
```typescript
// Apply Pencil values exactly
style={{
  fontFamily: 'Inter',
  fontSize: '14px',
  fontWeight: 500,        // Convert "500" → 500
  lineHeight: 1.5
}}
```

**If issues arise, see references/troubleshooting.md.**

### Phase 4: Verify

**Code Verification:**
- [ ] TypeScript compiles without errors
- [ ] All CSS variables converted to hex
- [ ] All spacing in exact pixels (no rem/em)
- [ ] Font specs match Pencil exactly

**Visual Verification:**
```typescript
// Compare screenshot with actual render
mcp__pencil__get_screenshot({
  filePath: "design.pen",
  nodeId: "targetNodeId"
})
```

Use DevTools to verify computed gap/padding values match Pencil exactly.

## Output Format

### 1. Analysis Summary
```markdown
## Pencil Analysis
- Structure: [key layout properties]
- Colors: [variable → hex mappings]
- Typography: [font specifications]
```

### 2. Implementation Code
```typescript
// Complete React/TypeScript component
// Inline styles for pixel precision
// Tailwind utilities only for w-full, flex, etc.
```

### 3. Verification Checklist
```markdown
✅ Completed items
⬜ Remaining verification steps
```

## Common Patterns

**Conditional Styling:**
```typescript
className={`border ${isSelected ? 'border-[#0084ff]' : 'border-[#3c3c3c'}`}
```

**Nested Layouts:**
```typescript
<div style={{ padding: '16px', gap: '12px', display: 'flex', flexDirection: 'column' }}>
  <div style={{ gap: '6px', display: 'flex', flexDirection: 'column' }}>
    {children}
  </div>
</div>
```

**Fixed Element Alignment:**
```typescript
<div style={{ paddingTop: '2px' }}>
  <Radio className="w-[18px] h-[18px]" />
</div>
<Label className="h-[18px] flex items-center">Label</Label>
```

**For complete working example, see references/complete-example.md.**

## Tool Usage Tips

**Batch Multiple Reads:**
```typescript
nodeIds: ["id1", "id2", "id3"]  // More efficient than separate calls
```

**Cache Variables:**
Call `get_variables()` once, reuse the mapping throughout conversion.

**Progressive Depth:**
Start with low readDepth for structure, then deep dive into specific nodes.

**Screenshot Last:**
Only for final verification—don't iterate with screenshots.

## Quick Reference Guide

**During conversion:**
1. Property mapping? → `references/quick-reference.md`
2. Need example? → `references/complete-example.md`
3. Something broken? → `references/troubleshooting.md`

**Key reminders:**
- No CSS variable references (always convert to hex)
- No Tailwind approximations (exact pixels only)
- Fixed heights for text alignment
- Font weights as numbers, not strings
