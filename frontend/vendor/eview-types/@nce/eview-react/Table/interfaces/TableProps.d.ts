import React, { CSSProperties } from 'react';
import ColumnProps from './ColumnProps';
import { SortOrder } from './ColumnProps';
import type { PagingProps } from '../../Paging';
import type { DialogProps } from '../../Dialog';
import type { PopupProps } from '../../Popup';
import type { HelpTipProps } from '../../HelpTip';
export default interface TableProps {
    id: string;
    /**
     * 行数据主键对应的列索引号，从0开始，不设置则主键默认使用行号
     */
    keyIndex: number;
    columns: ColumnProps[];
    dataset: any[];
    editOptions: any[];
    /**
     * 分页类型：数字列表或下拉框（'list'; 'select'）
     */
    pagingType: 'list' | 'select';
    /**
     *表头复选框排序使能
     */
    headerCheckBoxSortAllow: boolean;
    /**
     *表头复选框排序状态 'asc'升序;'desc'降序
     */
    headerCheckBoxSortStatus: string;
    /**
     * Table组件所在div的样式
     */
    className: string;
    /**
     * Table组件的高度，例如100、'100px'、'100%'，100和'100px'等效
     */
    height: number | string;
    /**
     * Table组件的宽度，例如100、'100px'、'100%'，100和'100px'等效
     */
    width: number | string;
    /**
     * 表选中行的自定义背景色class名称
     */
    selectedRowBgcCls: string;
    /**
     * 行选中事件回调
     *
     * @param {Object} row 行对象
     * @param {Object} e 原生点击事件
     */
    onRowClick: (row: any, event: Event) => void;
    /**
     * 鼠标移出行事件回调
     * @param {Object} e 原生点击事件
     */
    onRowMouseOut: (event: any) => void;
    /**
     * 鼠标移入行事件回调
     *
     * @param {Object} row 行对象
     * @param {Object} e 原生点击事件
     */
    onRowMouseOver: (event: any, row: any) => void;
    /**
     * 行点击事件的延时时间; 单位: 毫秒
     */
    rowClickDelay: number;
    /**
     * 行checkbox点击事件回调
     *
     * @param {Object} row 行对象
     * @param {Array} checkedRows 已勾选行数据主键数组，若设置keyIndex，数组元素为行数据中keyIndex列的数据，若不设置，则为行序号
     * @param {Object} e 原生点击事件
     */
    onRowCheck: (row: any, checkedRows: (string | number)[], e: Event) => void;
    /**
     * 表头勾选事件回调
     * @param {array} checkedRows 已勾选行数据主键数组，若设置keyIndex，数组元素为行数据中keyIndex列的数据，若不设置，则为行序号
     */
    onHeaderCheck: (checkedRows: any, checked: any, checkedRowsData: any) => void;
    /**
     * 可编辑单元，编辑时的回调，回调时机是onChange
     * @param {string} oldval The current value of the cell
     * @param {string} newval the new value of the cell
     * @param {Object} cell the current cell which is being edited
     * @param {Object} row the current row which is being edited
     * @param {Object} event native event
     */
    onEdit: (oldval: any, newVal: any, cell: any, row: any, event: Event) => void;
    /**
     * 可编辑单元失焦回调
     *
     * @param {Object} cell Cell that is being edited
     * @param {Object} row Row that is being edited
     */
    onEditingCellBlur: (cell: any, row: any) => void;
    /**
     * 分页点击回调
     *
     * @param {num} currentPage 当前页的序号
     */
    onPageChange: (currentPage: number) => void;
    /**
     * 分页尺寸点击回调
     *
     * @param {num} pageSize 表格页面可容纳行数
     */
    onPageSizeChange: (pageSize: number) => void;
    /**
     * 排序点击事件
     *
     * @param {string} sortColumn 排序列
     * @param {string} sortType 排序方式
     */
    onColumnSort: (sortColumn: any, sortType: SortOrder) => void;
    /**
     * 排序完成事件
     */
    onColumnSorted: (data: any) => void;
    /**
     * 延后onColumnSort的调用时机，用于解决部分后台排序的场景
     */
    delayOnColumnSort: boolean;
    /**
     * 返回展开行需要渲染的内容
     * @param {Object} row Table行对象
     */
    onRowExpend: (row: any) => React.ReactNode;
    /**
    * 点击展开的时候的回调函数
    *
    * @param {Object} row Table行对象
    */
    onRowExpendClick: (row: any) => void;
    /**
     * 开启分页
     */
    enablePagination?: boolean;
    /**
     * 开启自动分页
     */
    enableAutoPaging?: boolean;
    /**
     * Paging子组件的props; 详情请查阅Paging的API
     */
    pagingProps?: PagingProps;
    /**
     * 分页统计内容
     */
    pagingCountContent?: number | string | React.ReactNode;
    /**
     * 开启拖动
     */
    enableColumnDrag: boolean;
    /**
     * column的值要进行比价后，确认变化再更新
     */
    enableColumnCompareUpdate: boolean;
    /**
     * 开启列筛选
     */
    enableColumnFilter: boolean;
    /**
     * 列筛选的穿梭框是否支持搜索
     */
    enableColumnFilterSearch: boolean;
    /**
     * 自定义列筛选Dialog的zIndex
     */
    columnFilterZIndex: number;
    /**
     * columnFilterDialog的属性，可以传递Dialog组件支持的一些属性
     */
    columnFilterDialogProps: DialogProps;
    /**
     * 列复位到初始化的状态
     */
    enablColumnFilterResetInital?: boolean;
    /**
     * 开启多选框复选支持
     */
    enableCheckBox: boolean;
    /**
     * 是否禁用表头复选框
     */
    disableHeaderCheckbox: boolean;
    checkBoxPopupData: PopupProps;
    /**
     * 启用展开
     */
    enableRowExpand: boolean;
    /**
     * 允许同时展开多行
     */
    enableMulitiExpand: boolean;
    /**
    * 展开行的位置相对于表格冻结
    */
    enableRowExpandFreeze: boolean;
    /**
     * 表格可展示行数：是 pageSizeOptions 的第 0 个元素
     */
    pageSize: number;
    /**
     * 总记录数
     */
    recordCount: number;
    /**
     * 当前页码
     */
    currentPage: number;
    /**
     * 分页选项
     *
     * [10; 20; 50]
     */
    pageSizeOptions: number[];
    /**
     * 设置选中行，若设置keyIndex，则为行数据中keyIndex列的数据，若不设置，则为行序号
     */
    selectedRowIndex: number | number[];
    /**
     * 设置勾选的行，若设置keyIndex，则为行数据中keyIndex列的数据，若不设置，则为行序号
     */
    checkedRows: (string | number)[];
    /**
     * 配置了该属性之后，checkedRows属性更新时，会强制刷新，不会和上次值做对比
     */
    checkedRowsForceUpdate: boolean;
    /**
     * 保存跨页勾选
     */
    preserveCheckedRows: boolean;
    /**
     *  Right Click event callback.
     *
     * @param {Object} e Native click event
     * @param {Object} row Line object
     */
    onRowRightClick: (event: Event, row: any) => void;
    /**
     *  Display the item order Changer ; Item can be Up and Down .
     *  applicable only for for dataType: List
     *
     */
    itemOrderChanger: boolean;
    /**
     * Defines a class name mapping that will be applied to individual cells.
     * The property will be an Object of below format:
     * {
     *    [rowindex]: {
     *      [colkey]:[cellclassname]
     *    }
     * }
     * Note: The styles given in this classes should not try to override default styles which may upset the component as such. Be careful in using the property.
     */
    cellClassName: string;
    /**
     * Defines a style that will be applied to row tr tag.
     * The property will be an Object of below format:
     * {
     *    [rowindex]: {
     *     [rowStyle]
     *    }
     * }
     * Note: The styles given in this classes should not try to override default styles which may upset the component as such. Be careful in using the property.
     */
    customStyleRows: any;
    /**
     * 支持行多选
     */
    mutiSelectEnable: boolean;
    /**
     * 是否开启滑动多选checkbox，默认不开启
     */
    slidingMultipleSelected: boolean;
    /**
     *  If BM handle the complete soritng and dont want eview to do sorting.
     *  eview will not call "customSortFun" function also.
     */
    disableEviewSort: boolean;
    /**
     *  double Ok  Click event callback.
     *
     * @param {Object} HideRow  Hidden column name .
     * @param {Object} DisplayRow display column name.
     */
    onFilterOkClick: (hideRow: any, displayRow: any, columns: any) => boolean;
    /**
     *  if true custom sort will be enabled on the column .
     *  if false default sort will be enabled on the column.
     */
    customSortFun: (key: string | number, aValue: any, bValue: any, a?: any, b?: any) => number;
    /**
     * Defines the direction of the popup in the pagination page size selection control
     */
    paginationPopupDirection: 'top' | 'bottom';
    /**
     * @param {Object} proColumn details of column .
     */
    onColumnSizeChange: (proColumn: any, columns?: any) => void;
    /**
     * 设置行的单选或多选，默认值multi，enableCheckBox为true时，checkType才有效，checkType为multi时多选，checkType为msingle时单选，enableCheckBox为false时，checkType无效
     */
    checkType: 'multi' | 'single';
    /**
     *  Double check event on non Editable cell.
     */
    onDoubleClick: (evtRow: any, evtCell: any, nativeEvent: any) => void;
    /**
     * 可为table设置行属性
     */
    rowStyle: CSSProperties;
    /**
     *
     */
    expandedRow: (string | number)[];
    /**
     * 设置表格最大高度，超出高度显示滚动条
     */
    maxHeight: number | string;
    showEmptyImage: boolean;
    /**
     * To set the empty table message when data is not present in the table.
     */
    emptyTableMsg: string;
    /**
     * To set the disable row ids like [id1;id2;id3]
     */
    disableRowIds: (number | string)[];
    /**
     * disable对应行的checkbox
     */
    disableCheckboxIds: (number | string)[];
    /**
     * To set the enable custom ToolTip.
     */
    useCustomToolTip: boolean;
    /**
     * To set the custom ToolTip style; use with useCustomToolTip.
     */
    customToolTipStyle: CSSProperties;
    /**
     * check need to uupdate the column width dynamically
     */
    isRequiredToUpdateColumns: boolean;
    /**
     * To set width of select in paging
     */
    pagingSelectWidth: string;
    /**
     *  To set the total number of records
     */
    rowCount: number;
    /**
     *  on mouse dragon up for checkedrows function
     */
    onMouseUpOnRowForDrag: (startRow: any, endRow: any, checkedRows: any, e: any) => void;
    /**
     *table header checkbox sort call back function
     */
    onHeaderColumnSort: (event: any, sort: any) => void;
    /**
     *table minHeight
     */
    minHeight: number | string;
    /**
     * table cell click
     */
    onCellClick: (cCell: any, row: any, event: any) => void;
    /**
     * 二级表头合并配置，定义如下
     * 使用原有columns配置二级表头的第二行列头，使用此配置配置二级表头的第一行列头，未配置列合并的表头会自动进行行合并
     * groupHeaders； [{
     *   startColumnKey： String 指定开始列的Key;
     *   numberOfColumns: Number 合并列的个数;
     *   title: String 列头显示的文本
     * }，...]
     */
    groupHeaders: any[];
    /**
     * 三级表头
     */
    thirdHeaders: any[];
    /**
     * 是否显示行号列，行号列的排序不随表格排序变动，数字递增且不会被分页重置
     */
    enableShowLineNumber: boolean;
    /**
     * 滚动分页表格与显示行号一起使用时，需要传入当前展示的起始行数
     */
    startRowNumber: number;
    /**
     * 允许快捷方式打开过滤窗口
     */
    allowShortCutsFilter: boolean;
    /**
     * 是否滚动到展开行(表体Y轴出现滚动条属性生效)
     */
    isScrollToExpandedRow: boolean;
    /**
     *支持设置列选择弹出框挂载DOM的Id，默认挂在body上
     */
    columnFilterMountId: string;
    /**
     * Used in display of Total Record instead of recordCount.
     * Note : Total number of page calculation will be based on recordCount
     */
    recordCountDisp: string;
    /**
     * Callback for input box change in paging table
     */
    /**
     * set headerCheckBox status
     */
    headerCheckedStatus: 'empty' | 'all' | 'half';
    /**
     * 禁用表头双击恢复原始列宽
     */
    disableHeaderDoubClick: boolean;
    /**
     * 拖动列宽度时保持表格整体宽度不变
     */
    keepTableWidthAfterDragging: boolean;
    /**
     * 禁用行展开
     */
    disableRowExpand: (number | string)[];
    onColumnUpdate: () => void;
    splitPagination: boolean;
    onMouseMoveOnRowForDrag: (checkedRows: (string | number)[]) => void;
    textBlur: (cell: any, row: any) => void;
    enableOriginSort: boolean;
    showTooltipOverFlow: boolean;
    bordered: boolean;
    rowClickTriggerUpdate: boolean;
    /**
    * To set the disable columns ids like [id1,id2,id3]
    * disable的 column暂时没有特殊样式，但不可获取焦点
    */
    disableColumnsIds: (string | number)[];
    enableSelectedCount: boolean;
    /**
     * 表格选中行数，点击回调
     */
    onSelectedCountClick: () => void;
    selectedCount: string | number;
    checkCallBackOnce: any;
    handlePickColumns: any;
    enableFixColumnDrag?: boolean;
    enableZebraCrossing?: boolean;
    enableFixColumnZebraCrossingType?: 'normal' | 'panel' | 'dialog';
    enableSort?: boolean;
    /**
     * 冻结列的位置，左右
     */
    freezeColPosition: 'left' | 'right';
    onWindowResize?: () => void;
    onResetColumnFilter?: (optionalColumns: any, selectedColumns: any) => void;
    /**
     * 表头筛选面板关闭回调
     */
    onFilterClosed?: (event: Event, filterColumn: any) => void;
    /**
     * 开启鼠标控制横向滚动条
     */
    enableHorizontalScrollByWheel?: boolean;
    /**
     * 加载画面
     * */
    enableLoading?: boolean;
    /**
     * 勾选当前页，只有使用前台分页的时候才有效
     */
    enableCheckCurrentPage?: boolean;
    enableCompareRefresh?: boolean;
    enableColumnWidthFit?: boolean;
    enableLastColumnDrag?: boolean;
    /**
     * 是否支持虚拟滚动
     */
    virtualScroll?: boolean;
    virtualShowNum?: number;
    enableEmbeddedFilter?: boolean;
    /**
     * 当表格数据从有数据变成无数据时，是否要记录横向滚动条的位置
     */
    recodeTableScrollLeft?: boolean;
    /**
    * 自定义整个列定制弹窗
    */
    customColumnFilter: any;
    /**
     * 自定义列定制弹窗的内容
     */
    customColumnFilterDialog: any;
    /**
     * 表格resize的时候是否需要节流
     */
    resizeThrottle?: boolean;
    /**
     * 是否监听弹窗宽度的变化
     */
    resizeWithDialog?: boolean;
    /**
     * 传入目标元素，当其宽度变化，table执行resize
     */
    resizeWithTarget?: Element;
    /**
     * 开启swing表格相关的能
     * 1、双击表头自适应列宽
     */
    enableSwingAbility?: boolean;
    /**
     * 分割线双击自适应列宽是否开启
     */
    separatorDoubleClick?: boolean;
    /**
    * 分割线双击回调事件
    */
    onSeparatorDoubleClick?: any;
    /**
     *
     */
    collapseRowExpandWhileSort?: boolean;
    /**
     * 表格不在自动resize列宽
     */
    disableResize?: boolean;
    /**
     * 使用rowKey来指定data的主键
     */
    rowKey?: string;
    /**
     * 列定制的时候，优先显示自定义title
     */
    showCustomTitleInDoubleSelect?: boolean;
    onFilterIconClick?: any;
    onPageJumpStop?: any;
    /**
    * 场景：在table数据更新后，不希望在componentWillReceiveProps中销毁掉表头过滤面板
    * 在componentWillReceiveProps生命周期中将控制权移交给用户，决定是否执行表头过滤面板的更新
    * @returns boolean
    */
    shouldRenderColFilterOnUpdate?: (filterColumn: ColumnProps, columns: ColumnProps[], dataset: any) => boolean;
    /**
     * 自定义配置表头复选框右侧问号提示。
     */
    enableHeaderCheckboxHelpTip?: HelpTipProps;
    /**
     * 表头过滤面板是否挂载在body下。默认挂载在table内部
     */
    colFilterMountedOnBody?: boolean;
}
