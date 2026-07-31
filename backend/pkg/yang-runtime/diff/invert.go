package diff

// Invert 对变更集做回滚反算（CS-02）：ADD→按新增值删除、DELETE→按基线值
// 重建、MODIFY→新旧值互换。Path/SchemaPath 与顺序保持，输入切片不被改动。
// 反算结果经同一编码通道即得回滚报文；两次反算回到原集（幂等回环）。
func Invert(changes []Change) []Change {
	out := make([]Change, 0, len(changes))
	for _, c := range changes {
		inv := Change{Path: c.Path, SchemaPath: c.SchemaPath}
		switch c.Type {
		case AddChange:
			inv.Type = DeleteChange
			inv.OldValue = c.NewValue
		case DeleteChange:
			inv.Type = AddChange
			inv.NewValue = c.OldValue
		default: // ModifyChange 及未知类型按值互换处理
			inv.Type = c.Type
			inv.OldValue = c.NewValue
			inv.NewValue = c.OldValue
		}
		out = append(out, inv)
	}
	return out
}

// InvertResult 反算整个 DiffResult：Changes 逐条 Invert，Summary 重新累计
// （Adds/Deletes 互换、Modifies/Total 不变）。源结果不被改动。
func InvertResult(r *DiffResult) *DiffResult {
	out := NewDiffResult()
	for _, c := range Invert(r.Changes) {
		out.AddChange(c)
	}
	return out
}
