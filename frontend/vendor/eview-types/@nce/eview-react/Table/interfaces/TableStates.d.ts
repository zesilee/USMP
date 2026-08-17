import React from 'react';
import ColumnProps from './ColumnProps';
export interface ColumnState extends ColumnProps {
    widthFixed?: boolean;
    widthUnspecified?: boolean;
    text?: string;
}
export default interface TableStates {
    columns?: ColumnState[];
    /**
     * 表格渲染会根据data数据
     */
    data: any[];
    dataset: any[];
    dataValid: boolean;
    currentPage: number;
    pageSize: number;
    recordCount: number;
    checkedRows: (string | number)[];
    checkedRowsKey: Array<Array<number | string>>;
    displayColumnFilter: boolean;
    selectedRowIndex: (string | number)[];
    confirmButtonDisabled: boolean;
    resetButtonDisabled: boolean;
    expandedRow: (string | number)[];
    /**记录双向选择器的左侧内容 */
    doubleSelectEditLeftItems: any[];
    /**记录双向选择器的右侧内容 */
    doubleSelectEditRightItems: any[];
    filterDialog: React.ReactNode;
    enableColFilter: boolean;
    onHover: boolean;
    toolTipContent: string | React.ReactNode;
    isShowCheckBoxPopup: boolean;
    headerCheckBoxSortStatus: string;
    headerCheckBoxSortAllow: boolean;
    showMoveDown: boolean;
    showMoveUp: boolean;
    showThContentMenu: boolean;
    virtualStartIndex: number;
    trHeight: number;
    virtualShowNum: number;
}
