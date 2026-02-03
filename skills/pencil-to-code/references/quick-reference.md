# Quick Reference - Pencil to Code Conversion

## Essential Tool Calls (Always Start Here)

```typescript
// 1. Get structure
mcp__pencil__batch_get({
  filePath: "design.pen",
  nodeIds: ["targetNodeId"],
  readDepth: 4  // 2-3 for structure, 4-5 for detailed styles
})

// 2. Get color variables
mcp__pencil__get_variables({ filePath: "design.pen" })

// 3. Visual verification
mcp__pencil__get_screenshot({ filePath: "design.pen", nodeId: "targetNodeId" })
```

## Pencil → CSS Complete Mapping

### Layout & Positioning

| Pencil Property | CSS Equivalent | Example |
|----------------|----------------|---------|
| `layout: "vertical"` | `display: 'flex', flexDirection: 'column'` | `<div style={{ display: 'flex', flexDirection: 'column' }}>` |
| `layout: "horizontal"` | `display: 'flex', flexDirection: 'row'` | `<div style={{ display: 'flex', flexDirection: 'row' }}>` |
| `gap: 10` | `gap: '10px'` | `style={{ gap: '10px' }}` |
| `alignItems: "start"` | `alignItems: 'flex-start'` | `className="items-start"` |
| `alignItems: "center"` | `alignItems: 'center'` | `className="items-center"` |
| `alignItems: "end"` | `alignItems: 'flex-end'` | `className="items-end"` |
| `justifyContent: "start"` | `justifyContent: 'flex-start'` | `className="justify-start"` |
| `justifyContent: "center"` | `justifyContent: 'center'` | `className="justify-center"` |
| `justifyContent: "space-between"` | `justifyContent: 'space-between'` | `className="justify-between"` |

### Spacing

| Pencil Padding | CSS | Example |
|---------------|-----|---------|
| `padding: 10` | `padding: '10px'` | `style={{ padding: '10px' }}` |
| `padding: [10, 12]` | `padding: '10px 12px'` | `style={{ padding: '10px 12px' }}` |
| `padding: [10, 12, 14, 16]` | `padding: '10px 12px 14px 16px'` | `style={{ padding: '10px 12px 14px 16px' }}` |
| `padding: [2, 0, 0, 0]` | `paddingTop: '2px'` | `style={{ paddingTop: '2px' }}` |

### Dimensions

| Pencil | CSS | Example |
|--------|-----|---------|
| `width: 200` | `width: '200px'` | `style={{ width: '200px' }}` |
| `width: "fill_container"` | `width: '100%'` | `className="w-full"` |
| `height: 36` | `height: '36px'` | `style={{ height: '36px' }}` |
| `height: "hug_contents"` | `height: 'auto'` | `className="h-auto"` |

### Colors & Fills

| Pencil | CSS | Example |
|--------|-----|---------|
| `fill: "#FF0000"` | `backgroundColor: '#FF0000'` | `style={{ backgroundColor: '#FF0000' }}` |
| `fill: "$variable"` | **Convert to hex first!** | `style={{ backgroundColor: '#1e1e1e' }}` |
| `stroke: "#000000"` | `border: '1px solid #000000'` | `style={{ border: '1px solid #000000' }}` |
| `strokeThickness: 2` | `borderWidth: '2px'` | `style={{ borderWidth: '2px' }}` |

### Typography

| Pencil | CSS | Example |
|--------|-----|---------|
| `fontFamily: "Inter"` | `fontFamily: 'Inter'` | `style={{ fontFamily: 'Inter' }}` |
| `fontSize: 14` | `fontSize: '14px'` | `style={{ fontSize: '14px' }}` |
| `fontWeight: "500"` | `fontWeight: 500` | `style={{ fontWeight: 500 }}` |
| `lineHeight: 1.5` | `lineHeight: 1.5` | `style={{ lineHeight: 1.5 }}` |
| `textColor: "#FFFFFF"` | `color: '#FFFFFF'` | `style={{ color: '#FFFFFF' }}` |
| `textAlign: "center"` | `textAlign: 'center'` | `className="text-center"` |

### Borders & Effects

| Pencil | CSS | Example |
|--------|-----|---------|
| `cornerRadius: 6` | `borderRadius: '6px'` | `style={{ borderRadius: '6px' }}` |
| `cornerRadius: [4, 4, 0, 0]` | `borderRadius: '4px 4px 0 0'` | `style={{ borderRadius: '4px 4px 0 0' }}` |
| `opacity: 0.5` | `opacity: 0.5` | `style={{ opacity: 0.5 }}` |
| `shadow: {...}` | `boxShadow: '0 2px 8px rgba(0,0,0,0.1)'` | Convert shadow object to CSS |

### Special Cases

| Pencil | React/CSS | Notes |
|--------|-----------|-------|
| `ref: "ComponentId"` | Component instance | Read component structure with batch_get |
| `children: [...]` | Component slot override | Use descendants property |
| `visible: false` | `display: 'none'` or conditional render | `{visible && <Component />}` |

## Critical Rules (NEVER FORGET)

### 1. CSS Variables
```typescript
// ❌ WRONG - Tailwind doesn't support this
className="bg-[bg-primary]"

// ✅ CORRECT - Always convert to hex
// Step 1: mcp__pencil__get_variables()
// Step 2: Map variable to hex
className="bg-[#1e1e1e]"
```

### 2. Pixel Precision
```typescript
// ❌ WRONG - Approximation creates misalignment
className="space-y-2"  // 8px when Pencil is 6px

// ✅ CORRECT - Exact pixels
style={{ gap: '6px' }}
```

### 3. Alignment with Fixed Heights
```typescript
// ❌ WRONG - Text and checkbox won't align
<Checkbox className="w-[18px] h-[18px]" />
<Label>Text</Label>

// ✅ CORRECT - Fix label height to match checkbox
<Checkbox className="w-[18px] h-[18px]" />
<Label className="h-[18px] flex items-center">Text</Label>
```

### 4. Font Specifications
```typescript
// ✅ CORRECT - Match exactly
style={{
  fontFamily: 'Inter',      // Exact name
  fontSize: '14px',         // Add 'px'
  fontWeight: 500,          // Convert "500" to 500
  lineHeight: 1.5           // No units
}}
```

## Verification Checklist

Before considering the conversion complete:

- [ ] All CSS variables converted to hex codes
- [ ] All spacing in exact pixel values (no rem/em)
- [ ] All text elements have fixed heights for alignment
- [ ] Font specs match Pencil 100% (family, size, weight, line-height)
- [ ] TypeScript compiles without errors
- [ ] Screenshot comparison with Pencil design
- [ ] DevTools verification of gap/padding values

## Common Patterns

### Conditional Styling
```typescript
className={`border ${
  isSelected
    ? 'border-[#0084ff] bg-[#252525]'
    : 'border-[#3c3c3c] bg-transparent'
}`}
```

### Nested Layout
```typescript
<div style={{ padding: '16px', gap: '12px', display: 'flex', flexDirection: 'column' }}>
  <div style={{ gap: '6px', display: 'flex', flexDirection: 'column' }}>
    <div style={{ padding: '10px 12px', gap: '10px', display: 'flex' }}>
      {/* Content */}
    </div>
  </div>
</div>
```

### Fixed Element Alignment
```typescript
<div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
  <div style={{ paddingTop: '2px' }}>
    <Radio className="w-[18px] h-[18px]" />
  </div>
  <Label className="h-[18px] flex items-center" style={{ fontSize: '14px' }}>
    Label Text
  </Label>
</div>
```

## ReadDepth Strategy

| Goal | ReadDepth | Use Case |
|------|-----------|----------|
| Structure overview | 2-3 | Understanding component hierarchy |
| Detailed styling | 4-5 | Getting all style properties |
| Performance-sensitive | 1-2 | Quick checks, large files |
| Component internals | 5+ | Understanding complex nested structures |

## Performance Tips

1. **Batch multiple node reads**: `nodeIds: ["id1", "id2", "id3"]`
2. **Use appropriate readDepth**: Don't go deeper than needed
3. **Cache variables**: Call `get_variables()` once, reuse the mapping
4. **Screenshot last**: Only for final verification, not during iteration

---

**Quick Start:** Always run batch_get → get_variables → code → verify → screenshot
