package ygotbridge

// 测试专用别名：entry_*_test.go 自 pkg/yang-runtime/schema 整体迁入（阶段1.5），
// 别名让测试正文零改动、保持与原树的 diff 可读性。生产代码禁止用别名（只此
// _test.go 文件存在）。
import "github.com/leezesi/usmp/backend/pkg/yang-runtime/schema"

type (
	Schema        = schema.Schema
	DefaultSchema = schema.DefaultSchema
	Module        = schema.Module
	Node          = schema.Node
	NodeType      = schema.NodeType
	ContainerNode = schema.ContainerNode
	ListNode      = schema.ListNode
	LeafNode      = schema.LeafNode
	ChoiceNode    = schema.ChoiceNode
	CaseNode      = schema.CaseNode
)

const (
	LeafTypeBoolean   = schema.LeafTypeBoolean
	LeafTypeInt8      = schema.LeafTypeInt8
	LeafTypeInt16     = schema.LeafTypeInt16
	LeafTypeInt32     = schema.LeafTypeInt32
	LeafTypeInt64     = schema.LeafTypeInt64
	LeafTypeUint8     = schema.LeafTypeUint8
	LeafTypeUint16    = schema.LeafTypeUint16
	LeafTypeUint32    = schema.LeafTypeUint32
	LeafTypeUint64    = schema.LeafTypeUint64
	LeafTypeString    = schema.LeafTypeString
	LeafTypeEnum      = schema.LeafTypeEnum
	LeafTypeEmpty     = schema.LeafTypeEmpty
	LeafTypeDecimal64 = schema.LeafTypeDecimal64
	LeafTypeBits      = schema.LeafTypeBits

	ContainerNodeType = schema.ContainerNodeType
	ListNodeType      = schema.ListNodeType
	LeafNodeType      = schema.LeafNodeType
	ChoiceNodeType    = schema.ChoiceNodeType
	CaseNodeType      = schema.CaseNodeType
)

var (
	NewSchema    = schema.NewSchema
	NewContainer = schema.NewContainer
	NewModule    = schema.NewModule
)
