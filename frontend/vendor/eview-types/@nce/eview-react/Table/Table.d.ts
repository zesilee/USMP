/// <reference types="node" />
import React, { Component } from 'react';
import Dialog from '../Dialog';
import TableProps from './interfaces/TableProps';
import TableStates from './interfaces/TableStates';
import type { SortOrder } from './interfaces/ColumnProps';
import type { ColumnState } from './interfaces/TableStates';
import VirtualScroll from '../VirtualScroll';
export default class Table extends Component<TableProps, TableStates> {
    static defaultProps: Partial<TableProps>;
    static ColumnRenderType: {
        CHECK_BOX: string;
        CHECK_BOX_GROUP: string;
        DATE_PICKER: string;
        TIME_SELECTOR: string;
        PROGRESS_BAR: string;
        RADIO_GROUP: string;
        SELECT: string;
        INPUT_SELECT: string;
        TEXT_FIELD: string;
        CUSTOM: string;
    };
    static contextType: React.Context<import("react-intl").IntlShape>;
    static CHECK_BOX_WIDTH: number;
    static CHECK_BOX_WIDTH_PLUS_PADDING: number;
    static COLUMN_MIN_WIDTH: number;
    static EXPEND_WIDTH: number;
    static OPERATION_ICON_WIDTH: number;
    static OPERATION_PADDING: number;
    static LINE_NUMBER_WIDTH: number;
    private isMoved;
    isEmptySelect: boolean;
    isHeaderCheckBoxSelected: boolean;
    columnDragIsFired: boolean;
    clicktype: string;
    selectedRowBgcCls: string;
    collerColFilterDiv: boolean;
    filterColumn: string;
    filterPopId: string;
    dbs: any;
    refreshFlag: boolean;
    id: string;
    unstate: any;
    isScrollToExpandedRow: boolean;
    isSelectRowIndexSet: boolean;
    freezeTable: boolean;
    cellStyle: React.CSSProperties;
    dblClkEditCell: any;
    dblClkEditRow: any;
    dblClkEditElement: any;
    preEditCell: any;
    preEditCellEle: any;
    preEditRow: any;
    cellEditing: any;
    dblClkEditNativeEvent: any;
    apiEditCell: (string | number)[];
    apiEditRow: any;
    propsChanged: boolean;
    focusedCell: any;
    currentTable: any;
    firstShiftSelected: any;
    sorted: SortOrder;
    clickCount: number;
    singleClickTimer: NodeJS.Timeout;
    didComponentMount: boolean;
    selectedColumnsForReset: ColumnState[];
    optionalColumnsForReset: any;
    showFilter: boolean;
    originalColumns: any;
    dragOnRow: boolean;
    lastRowMovedOn: any;
    focusRowId: string;
    lastSortColumnCid: any;
    lastSortColumnKey: string;
    changeTableWidth: boolean;
    textField: any;
    textFieldFocusCell: any;
    textFieldFocusRow: any;
    timerCell: any;
    totalWidth: number;
    slidingMultDragStartRow: any;
    slidingMultLastRowMovedOn: any;
    onMouseDown: boolean;
    isAbsDrag: boolean;
    mouseDirectionChange: boolean;
    slidingMultipleCheckRows: (string | number)[];
    clickdata: any;
    clickcolumnData: any;
    columnWidth: any;
    Header: any;
    table: any;
    table1: any;
    container: any;
    tableDiv: any;
    tableDiv1: any;
    op: any;
    headerTable1: any;
    headerTable: any;
    baseDiv: HTMLDivElement | HTMLElement;
    filterIcon: any;
    filterDiv: HTMLDivElement;
    filterButton: any;
    checkBoxPop: any;
    checkBoxPopupDiv: HTMLElement;
    customTip: any;
    cellReference: HTMLElement;
    headerTableDiv: HTMLDivElement;
    thPopRef: any;
    thContextDom: any;
    selectedColumns: any;
    optionalColumns: any;
    editComponent: any;
    draging: any;
    dragingLeft: number;
    dragingLeftFreezed: number;
    downedHeader: any;
    dragingDom: any;
    dragingColumn: any;
    pagePaneContainer: any;
    isClickReset: boolean;
    tableHeader: any;
    dragStartRow: any;
    textFieldFalg: any;
    pagePane: any;
    windowResize: boolean;
    isResizeWindow: boolean;
    headerTr: HTMLTableRowElement;
    endOfTable: HTMLDivElement;
    noDataDiv: HTMLDivElement;
    dia: Dialog;
    movePx: number;
    lastCid: number;
    dragingColumnIndex: number;
    isLastColFilterEnable: boolean;
    checkedRowState: any;
    startSelectRow: any;
    endSelectRow: any;
    formatMessage: any;
    delayTimer: any;
    isInTipBox: boolean;
    onWheelFlag: boolean;
    onWheelTarget: any;
    contentHeight: any;
    contentMinHeight: any;
    contentMaxHeight: any;
    contextMenuColumn: any;
    dragingColumnWidth: any;
    isTableDivFirstScroll: boolean;
    isHeaderFirstScroll: boolean;
    virtualScroll: VirtualScroll;
    resizeFlag: boolean;
    mouseNearGap: boolean;
    hasVScrollLast: boolean;
    resizeObserver: ResizeObserver;
    dialogWidth: number;
    dialogHeight: string | number;
    dialogPosition: any;
    resizeColumnFlag: boolean;
    parentDialog: Element;
    swingShiftStartRow: any;
    swingSlidingStartRowIsChecked: any;
    notTriggerClick: boolean;
    recordOldDialogWidth: string;
    updateFilterPositionFlag: boolean;
    /**
     * 为classname属性增加class
     *
     * @static
     * @param {String} classes DOM对象的class属性的字符串
     * @param {String} addClass 需要加的样式
     *
     * @memberof Table
     */
    static addClass(classes: any, addClass: any): any;
    /**
     * 为classname属性增加class
     *
     * @static
     * @param {String} classes DOM对象的class属性的字符串
     * @param {String} removeClass 需要删的样式
     *
     * @memberof Table
     */
    static removeClass(classes: any, removeClass: any): any;
    constructor(props: any, context: any);
    /** 检查PageSize是否合法 */
    checkPageSize(pageSize: any): any;
    isElementTableContainer: (element: any) => boolean;
    /**
     * 处理右键的事件
     * @param event
     */
    _handleContextMenu: (event: any) => void;
    /**
     * User will call this API to reset the original positions .
     */
    reSetcolumnSize: () => void;
    setScrollPositionToColumn(colName: any): void;
    /**
     * 设置CheckedRows
     * @param checedkRows
     */
    setCheckedRows: (checedkRows: Array<number | string>) => void;
    setCurrentPage: (currentPage: any) => void;
    /**
     * 设置选中行
     */
    setSelectedRowIndex: (selectedRowIndex: any) => void;
    /**
     * 设置某一列列宽根据表头和内容自适应，将列宽调整为最大的
     * @param id
     * @returns
     */
    setSizeColumnFit: (cid: any) => ColumnState[];
    /**
     * public
     * 设置多列列宽根据表头和内容自适应，将列宽调整为最大的
     * @param id
     */
    setColumnsFit(cid?: any): void;
    /**
     * id是索引，从0开始
     * @param id
     * @param enableScrollOnExpand 默认是行展开后，此api失效。此参数设置为true，行展开后，依然支持滚动。
     */
    setScrollBarAt: (id: any, enableScrollOnExpand?: boolean) => void;
    /**
     *
     * @param num 行数，从1开始
     * @returns
     */
    setVirtualScrollBarAt: (num: any) => void;
    /**
     * 设置虚拟滚动的显示条数
     * @param num
     */
    setVirtualShowNum: (num: any) => void;
    getSizeOfText: (value: any, fontSize: any, className: any) => number;
    updateColumnHeders(props: any): any[];
    registerContextMenuListner(): void;
    unRegisterContextMenuListner(): void;
    /**
     *  调整设置按钮
     */
    adjustSettingIcon(): void;
    /**
     * 根据表格的列是否要更新，返回对应的columns属性
     * @param columns
     * @param update
     * @returns
     */
    matchNextPropsColumnsWithStateColumns: (prevColumns: any, columns: any, update: any, compareUpdate: any) => any;
    saveClumnsForReset: (columns: any) => void;
    /**
     * 当表格数据从有数据变成无数据时，要记录横向滚动条的位置
     */
    recodeTableScrollLeft: (nextProps: any) => void;
    componentWillReceiveProps(nextProps: any): void;
    componentDidMount(): void;
    /**
     * 存在列没有宽度
     * @returns
     */
    isUpdateWindowResize: () => boolean;
    monitorScrollUpdateFilterPosition: (e: any) => void;
    /**
     * 设置过滤面板的位置
     */
    setFilterPosition(): void;
    /**
     * DidUpdate是在render函数之后调用的
     */
    componentDidUpdate(): void;
    /**
     * 处理上一次的排序，恢复被冲掉的数据
     * @param column
     * @param columns
     * @param nextProps
     * @param data
     */
    handleLastSort: (column: any, columns: any, nextProps: any, data: any) => any;
    /**
     * 设置Tipbox的位置
     */
    updateToolTip(): void;
    updateColumnWidth(isIntialize: any): void;
    getUnSpecifiedColumnsWidth(tableWidth: any): number;
    componentWillUnmount(): void;
    getCurrentPage(): number;
    getPageSize(): number;
    getSortColumn(): any;
    getSortType(): any;
    getKeySortColumn(): ColumnState;
    /**
     * 获取单选行数据
     *
     * @returns {object[]}
     *
     * @memberof Table
     */
    getSelectedRowData(): any[];
    /**
     * 获取是否开启滑动开关
     *
     * @returns {bool}
     *
     * @memberof Table
     */
    getRowSlidingMultipleSelectedFlag(): boolean;
    /**
     * 返回选中行，只返回单行
     * @returns
     */
    getSelectedRowIndex(): string | number | (string | number)[];
    /**
     * 返回选中行
     * @returns
     */
    getSelectedRowsIndex(): (string | number)[];
    /**
     * 获取多选行
     * @returns {object[]}
     * @memberof Table
     */
    getCheckedRowsData(): any[];
    /**
     * 返回勾选的行的index，支持跨页勾选
     * @returns
     */
    getCheckedRowsIndexes(preserveCheckedRows?: any): (string | number)[] | (string | number)[][];
    /**
     * 获取当前数据集
     * @returns {object[]}
     * @memberof Table
     */
    getDataset(): any[];
    /**
     * 获取详细的数据集
     * @returns {object[]}
     * @memberof Table
     */
    getData(): any[];
    /**
     * 获取需要导出的数据集，包括表头和数据
     */
    getExportDataset(): any[];
    /**
     * 获取当前数据的columns相关信息
     */
    getColumns(): ColumnState[];
    /**
     * 更新当前表格的列，只能改变列的状态，比如排序、筛选等，不能修改列的显示数量
     * columns[i]中key或者title属性是必须的
     */
    updateColumns(columns: any): void;
    /**
     * 获取原始数据集
     *
     * @returns {object[]}
     *
     * @memberof Table
     */
    getOriginDataset(): any[];
    /**
     * 获取编辑的行数据
     *
     * @returns {array}
     *
     * @memberof Table
     */
    getEditData(): any;
    setRowEditable(id: any, columns: any): void;
    getCheckboxThWidth: () => number;
    getNearColumn(offsetLeft: number, offsetLeftFreezed: number): any;
    getColumn(cid: any): any;
    compileCellStyle: (cellClassNameProp: any) => any;
    filterAndSortColumns(data: any): any;
    /**
     * 获取不同浏览器的滚动条的宽度
     */
    getBrowserScrollbarWidth(): 8 | 12;
    genRow(row: any, editRow: any, freezeCol: any, freezeTable: any, lineNo: any): any[];
    handleMouseDownOnTD: (event: any) => void;
    getRealCellTip(cell: any, row: any, option: any): any;
    /**
     * 延迟设置onHover状态，Hover后500ms再显示Tips
     * @param onHover
     * @param toolTipContent
     * @returns
     */
    setDelayOnHover: (onHover: boolean, toolTipContent: string, time?: any) => void;
    clearDelayTimer: () => void;
    handleScroll: () => void;
    /**
   * 虚拟滚动的回调
   * @param startIndex
   */
    handleVirtualScroll: (startIndex: any, scrollLeft: any, event: any) => void;
    handleTipBoxMouseLeave: (e: any) => void;
    handleTipBoxMouseEnter: (e: any) => void;
    /**
     * 处理td的MouseEnter事件
     * @param e
     * @param row
     * @param cell
     * @returns
     */
    handleMouseEnter: (e: any, row: any, cell: any) => void;
    removeToolTip: (row: any, e: any) => void;
    /**
     *
     * @param colTitle
     * @param index
     * @returns
     */
    getCurrentColumnWidth(colTitle: any, index: any): number;
    genColumns(freezeCol: any, freezeColPosition: any, genFreezeTable: any, genHeadCol: any): React.JSX.Element;
    getColumnFilter(obj: any): React.JSX.Element;
    handleColFilterKeyPress: (column: any) => (event: any) => void;
    handleColFilter: (column: any) => (e: any) => void;
    getTableFilterIconClass: (column: any) => string;
    getDisplayColumns(freezeCol: any): ColumnState[];
    /**
     * 是否有垂直的滚动条
     * @returns
     */
    hasVScroll: () => boolean;
    genHeaders(freezeCol: any, freezeTable: any): React.JSX.Element;
    handleCheckBoxSortClick: (sort: any) => (event: any) => void;
    handleCheckBoxSortKeyDown: (headerSortInfos: any) => (event: any) => void;
    handleClickCheckBoxPopupIcon: (e: any) => void;
    /**
     * 获取表头全选checkbox的状态，支持手动设置
     * @param data
     * @param checkedRows
     * @returns
     */
    getCheckboxStatus(data: any, checkedRows: any): "all" | "empty" | "half";
    genBody(freezeCol: any, freezeTable: any, freezeColPosition: any, genFreezeTable: any): React.JSX.Element;
    handleMouseDownOnRowForDrag: (row: any) => (e: any) => void;
    swingDragAndCheckedRow(row: any, e: any, isChecked: any): void;
    dragAndCheckedRow(row: any): void;
    handleMouseMoveOnRowForDrag: (row: any) => (e: any) => void;
    handleMouseUpOnRowForDrag: () => (e: any) => void;
    handleKeyDownForCopy: (event: any) => void;
    /**
     *  To adjust the width of the last "VISIBLE" column with "width undefined or in %" to add the missed pixels(due to parseInt rounding up) to it
     *  so table width will remain the same
     */
    adjustLastColWidth: () => void;
    /**
     *  根据列宽来获取整个表格的宽度，返回的是列宽和
     */
    getTableWidth(): number;
    /**
     * checkbox下拉面板按键回调
     * @param {*} event
     */
    handleOptionKeyDown: (event: any) => void;
    handleKeyDownCheckBoxPopupIcon: (event: any) => void;
    handleCheckBoxPopupItemClicked: (item: any) => void;
    getThRowSpan(): 1 | 2 | 3;
    /**
     * 为表头添加行号列
     * @param headerRows 列头的dom元素集合
     * @param rIndex 元素个数
     * @param freezeCol 是否为冻结列表格
     */
    addLineNumber: (headerRows: any, rIndex: any, freezeCol: any, left: any) => void;
    /**
     * 为表头添加复选框
     * @param headerRows 列头的dom元素集合
     * @param rIndex 元素个数
     * @param freezeCol 是否为冻结列表格
     */
    addHeaderCheckBox: (headerRows: any, rIndex: any, freezeCol: any, left: any) => void;
    /**
     * 为表头添加展开列
     * @param headerRows 列头的dom元素集合
     * @param rIndex 元素个数
     * @param freezeCol 是否为冻结列表格
     */
    addHeaderRowExpand: (headerRows: any, rIndex: any, freezeCol: any, left: any) => void;
    getTableHeightFromProps: () => any;
    /**
     * 如果设置了maxHeight属性 设置表格的高度为固定宽度，而不是100%
     * maxHeight的向下继承是有问题的，表格的最外层设置了maxHeight，不能被子级的div继承
     */
    setTableHeight(): void;
    styleToNumber: (height: any) => number;
    /**
     * 计算表格纯内容（不包括表头和分页）的高度
     * 目前不兼容表格高度自定义的情况
     * @param height
     * @returns
     */
    calcHeight: (height: any) => string;
    genTable(): React.JSX.Element;
    genNoDataTable(): React.JSX.Element;
    /**
     * 获取Tips
     * @param cell
     * @param row
     * @param option
     * @returns
     */
    getEditCellTip(cell: any, row: any, option: any): any;
    /**
     * 根据type参数，生成编辑表格单元，当前支持的类型包括TextField,DATE_PICKER,TimeSelector,PROGRESS_BAR等
     * @param type
     * @param cell
     * @param options
     * @param row
     * @param editOptions
     */
    genEditCell(type: any, cell: any, options: any, row: any, editOptions: any): any;
    handleMoveRow: (rowId: any, isUp: any) => void;
    /**
     * 生成单元格
     * @param type
     * @param cell
     * @param options
     * @param row
     * @param editOptions
     * @returns
     */
    genCell(type: any, cell: any, options: any, row: any, editOptions: any): any;
    checkIsEditable: (selectedColumns: any, optionalColumns: any) => boolean;
    changeResetButtonState: (array: any) => void;
    /**
     *
     * @param arrary 右侧的项目，不是左侧的
     * @returns
     */
    onMoveToRight: (arrary: any) => void;
    handleRightOrderChange: (array: any) => void;
    /**
     *
     * @param arrary 移动到左侧的子项
     * @param isAll 全部组件都移动到左侧
     * @param selectedItems 右侧剩余的子项
     * @returns
     */
    onMoveToLeft: (arrary: any, isAll: any, selectedItems: any) => void;
    /**
   * 当column.title传入是非字符串时,使用column.titleComponentToText字段带图
   * @returns
   */
    getComponentToText(column: any): any;
    /**
     * 生成列筛选面板
     * @returns
     */
    genColumnSwitch(): any;
    handleCoverReset: () => void;
    handleKeyUpFake: () => void;
    /**
     *  全局监听的事件,handleEditableCellClick函数调用的时候会绑定
     *  handleOutsideAction调用的时候会解绑
     *,当this.textField不为空的时刻，就会截取click事件,交给这个函数处理
     * textfield失焦的时候才应该执行，focus的时候不应该执行
     * preEditCell
     * preEditCellEle
     * @param event
     * @param textBlur
     */
    handleOutsideAction: (event: any, textBlur?: any) => boolean;
    /**
     * 重置编辑态的表格单元
     * @param edtCell 编辑态表格单元
     * @param edtRow  编辑态表格单元所在的行
     * all,所有的编辑单元格都复位
     */
    resetEditableCell: (edtCell: any, edtRow: any, all: any, isBlur: any) => void;
    getFocusdTd: (evtCell: any, evtRow: any, evt: any, header: any) => void;
    getColumnFocused: (rowID: any, colID: any, table: any, evt: any) => void;
    getHeaderFocused: (rowID: any, colID: any, table: any, evt: any) => void;
    /**
     * 表格的td元素，键盘事件回调
     * @param {*} evtCell
     * @param {*} evtRow
     * @param {*} event
     */
    handleKeyPressOnTD: (evtCell: any, evtRow: any, event: any) => void;
    handleClearSelectedRowBG: () => void;
    handleSelectedRowBG: (currentFocus: any) => void;
    handleKeyChangeColumnWidth(changeColumn: any, step?: number): void;
    handleKeyPressOnTH: (column: any, event: any) => void;
    focusOnColFilterIcon: (event: any) => void;
    focusAndScroll: (focusElement: any, focusingHeader?: any) => void;
    focusOnTableHeader: () => void;
    handleColumnFilterOnKeyPress: (event: any) => void;
    handleCellDoubleClick: (evtCell: any, evtRow: any, event: any) => void;
    /**
     * 处理表格点击事件,冒泡，所以当表格单元为TextField时，不会冒泡给td，处理这个事件
     * @param evtCell
     * @param evtRow
     * @param event
     */
    handleCellClick: (evtCell: any, evtRow: any, event: any) => void;
    /**
     * 处理可编辑的表格单元点击事件
     *
     * @param evtCell
     * @param evtRow
     * @param event
     */
    handleEditableCellClick: (evtCell: any, evtRow: any, event: any) => void;
    handleemptyRightClick: (evtCell: any) => void;
    /**
     * 鼠标进入非表头、行区域时，比如行展开区域等，不执行回调
     */
    handleTDThMouseLeave: () => void;
    handleColumnRightClick: (evtCell: any, event: any) => void;
    handleThContextMenu: (column: any, event: any) => void;
    handleContextMenuHide: () => void;
    handleThContextMenuClick: (data: any, event: any) => void;
    handleTrRightClick: (row: any, e: any) => void;
    getCheckStartAndEndIndex: (first: any, last: any, data: any) => any[];
    /**
     * 处理行勾选，shiftKey？？
     * @param row
     * @returns
     */
    handleCheck: (row: any) => (propsVal: any, isChecked: any, e: any, isRowClick: any) => any;
    /**
     * 单选框勾选
     * @param row
     * @returns
     */
    handleRadioCheck: (row: any) => (val: any, e: any) => void;
    /**
     * 判断id是否在当前页中;
     * @param id
     * @returns
     */
    isIDInCurrentPage: (id: any, data: any) => boolean;
    /**
     * 头部全选checkbox事件处理
     * @param value
     * @param checked
     * @param event
     */
    handleHeaderCheck: (value: any, checked: any, event: any) => void;
    handlePageChange: (currentPage: any) => any;
    handlePageSizeChange: (pageSize: any) => void;
    getKeyAndSort(): {
        key: any;
        sort: any;
    };
    handleSwingCheck: (row: any, e: any) => boolean;
    /**
     * 行点击事件
     * @param row
     * @returns
     */
    handleRowClick: (row: any) => (e: any) => boolean;
    handleSort: (column: any) => () => void;
    /**
     * 可以主动调用，resize表格的方法
     * 表格监听父容器变化的回调，不会保留用户拖动修改后的列宽
     * @param e
     */
    handleParentResize: (e?: any, disableReset?: boolean) => void;
    /**
     * 点击过滤面板外侧，隐藏过滤面板
     * @param e
     */
    handleClickOutSideFilter: (e: any) => void;
    isChild: (element: any) => boolean;
    /**
     * window size改变的时候，刷新表格的列宽，根据props中的初始列宽来刷新，而不是state中的列宽
     * 在需要主动刷新列宽的时候，也可调用改函数（仅限函数内部）
     * @param e
     * @param isWindowResize
     */
    handleWindowResize: (e?: any, isWindowResize?: any, disableReset?: boolean) => void;
    onTHDoubleClickColWidthRestore: (col: any, e: any) => void;
    handleTheadKeyDown: (e: any) => void;
    getDragColumnWidthAndFix: (columns: any) => any;
    /**
     *
     * @param e
     * @returns
     */
    isFreezedColumnsWidthOver: (columns: any) => boolean;
    handleTableKeyDownCtrl: (e: any) => void;
    handleMouseDown: (e: any) => void;
    /**
     * 判读鼠标是否靠近
     * @param e
     * @returns
     */
    handleMouseMove: (e: any) => void;
    /**
     * 全局捕获事件，
     * 表头拖动时改变列宽时，处理相关的事件
     * @param e
     */
    handleGlobalMouseMove: (e: any) => void;
    /**
     * 用于设置列拖拽改变后的宽度
     * @param e
     * @returns
     */
    handleMouseUp: (e: any) => void;
    handleDrop: (event: any) => void;
    handleDragStart: (event: any) => void;
    handleExpendOnKeyPress: (row: any) => (event: any) => void;
    /**
     * 表格展开按钮点击回调
     * @param row
     */
    handleExpend: (row: any) => (e: any) => void;
    handleTextEdit: (cell: any, row: any) => (value: any) => void;
    handleTextFocus: (cell: any, row: any) => () => void;
    handleTextBlur: (cell: any, row: any) => (event: any) => void;
    handleSelectEdit: (cell: any, row: any) => (value: any) => void;
    handleInputSelectEdit: (cell: any, row: any) => (value: any) => void;
    handleRadioEdit: (cell: any, row: any) => (value: any) => void;
    handleCheckBoxEdit: (cell: any, row: any) => (_: any, value: any) => void;
    handleCheckBoxGroupEdit: (cell: any, row: any) => (value: any) => void;
    handleDateEdit: (cell: any, row: any, type: any) => (_: any, value: any) => void;
    hanldleTimeSelectorEdit: (cell: any, row: any) => (value: any) => void;
    /**
     * 复位表格的横向滚动条
     */
    resetScollbar(): void;
    handleOuterScroll: () => void;
    /**
    * 表格滚动回调事件，同步改变表头的scrollLeft
    * 残疾人下表头焦点切换会触发滚动
    * @param e
    */
    handleHeaderTableDivScroll: (e: any) => void;
    /**
     * 表格滚动回调事件，同步改变表头的scrollLeft
     * 注意：改变表头的scrollLeft会触发表头的header
     * @param e
     */
    handleTableDivScroll: (e: any, virtualScrollLeft?: number) => void;
    /**
     * 如果表格没有纵向滚动条，则改变横向滚动条
     * @param e
     */
    handleGlobalWheel: (e: any) => boolean;
    handleTableDivWheel: (e: any) => boolean;
    handleTableInnerWheel: (e: any) => boolean;
    handlePickColumns: (event: any) => void;
    /**
     * 处理列过滤面板关闭事件
     */
    handleCloseColumnFilter: () => void;
    handleDoubleSelectEditItems: (leftItems: any, rightItems: any) => void;
    handleColumnValidation: (item: any, value: any) => void;
    handleColumnFilterDialogResize: (obj: any) => void;
    /**
     * 获取表格容器的宽度，用于计算列宽等,
     * 表格的异常出现的滚动条可以定位到这个函数
     * 宽度优先级：tableDiv>baseDiv>外层div
     */
    getTableContainerWidth(): any;
    /**
     * 获取表格功能列的宽度
     * @returns
     */
    getTableFunctionColumnsWidth: () => number;
    /**
     * 1、当属性设置的column.width之和不是100%的时候， getColumnWidth计算的表格列宽是不对的，所以需要修正下
     * 2、拖拽列需要的是精准的宽度，而不能是比例宽度
     * 只修正col的width之和小于tableWidth的情况
     * @param columns
     */
    fixColumnWidth(columns: any): void;
    /**
     * To get the column width as user specification ,othrwise we are calculte the table width and get  the column unspecififed width.
     * @param {*} colName
     */
    getColumnWidth(colName: any, nextProps?: any): string;
    /**
     * 获取未指定宽度的列的列宽
     * @param {*} tableWidth
     */
    getUnspecifiedColumnWidth(tableWidth: any, props: any): number;
    handleResetColumnFilter: () => void;
    handleOkColumnFilter: () => void;
    handleColumnSort: (columnArr: any) => void;
    handleColumnFilter: (optionalColumnsForReset?: any, selectedColumnsForReset?: any) => void;
    handleFilterColumnsWidth: (columns: any) => any;
    render(): React.JSX.Element;
}
