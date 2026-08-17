import React, { Component } from 'react';
export interface TableHeaderProps {
    freezeConfig: any;
    groupHeaders: any;
    thirdHeaders?: any;
    columns: any;
    enableRowExpand: any;
    enableCheckBox: any;
    handleMouseMove: any;
    handleMouseDown: any;
    addHeaderCheckBox: any;
    addHeaderRowExpand: any;
    getCheckboxThWidth: any;
    enableShowLineNumber: any;
    addLineNumber: any;
    tableDiv: any;
    getBrowserScrollbarWidth: any;
    handleSort: any;
    parentComponent: any;
}
export default class TableHeader extends Component<TableHeaderProps> {
    static CHECK_BOX_WIDTH: number;
    static CHECK_BOX_WIDTH_PLUS_PADDING: number;
    static COLUMN_MIN_WIDTH: number;
    static EXPEND_WIDTH: number;
    static OPERATION_ICON_WIDTH: number;
    static OPERATION_PADDING: number;
    static LINE_NUMBER_WIDTH: number;
    Header: HTMLTableHeaderCellElement;
    constructor(props: any);
    /**
     * 创建二级表头的组件,firstRow是最上面一行，secondRow是下面一行
     * 暂时不支持固定列
     * @param groupHeaders 二级表头配置
     * @param columns 原始列配置
     * @param freezeCol 冻结列表格
     * @return [[...],[...]] 表头的二维数组结构
     */
    createSecondHeadersModel: (groupHeaders: any, columns: any, freezeCol: any, isThird?: any, exclude?: any) => any[][];
    /**
     * 处理列头的基础样式
     * @param column 列配置
     * @returns {[string]}
     */
    getHeaderClass: (column: any) => string[];
    /**
     * 过滤掉隐藏列
     * @param columns
     * @param freezeCol
     * @param freezeTable
     * @returns {*[]}
     */
    getDisplayColumns: (columns: any, freezeCol: any, freezeTable: any, freezeColPosition: any) => any;
    /**
     * 获取二级表头
     * @return {*} 表头的html元素片段
     */
    genMultiHeaders(): React.JSX.Element;
    render(): React.JSX.Element;
}
/**
 * 二级表头需要重置freezeCol来确保如果合并列中某一列配置了冻结列，那么该合并列整体都会被冻结
 * @param columns 列配置信息
 * @param groupHeaders 二级表头配置信息
 */
export declare function resetFreezeColumns(columns: any, groupHeaders: any): void;
/**
 * 获取列与列之间的分隔符
 * @param index 列下标
 * @param column 列信息
 * @param enableRowExpand 是否支持展开
 * @param enableCheckBox 是否含有checkBox
 * @returns {*}
 */
export declare function genColumnSeparator(index: any, column: any, enableRowExpand: any, enableCheckBox: any, enableShowLineNumber?: boolean, isLast?: boolean, that?: any): React.JSX.Element;
/**
 * 冻结列时调整原始表格的列顺序，如果freezeColPosition设置的是left，则把所有冻结列排列动队列前
 * 否则排列子队列尾部
 * @param columns 列配置
 * @param freezeTable 是否冻结列
 * @param freezeColPosition 冻结列是显示在表格的左边还是右边
 * @returns {[]}
 */
export declare function handleColumnSequence(columns: any, freezeColPosition: any): any[];
