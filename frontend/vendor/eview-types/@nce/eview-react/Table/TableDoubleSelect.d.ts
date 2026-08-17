import React, { Component } from 'react';
export interface TableDoubleSelectProps {
    leftHeader: any;
    rightHeader: any;
    leftContentHeader: any;
    rightContentHeader: any;
    width: any;
    model: any;
    isClickReset: any;
    items: any;
    selectedItems: any;
    forwardRef: any;
    itemOrderChanger: any;
    getSelectedData: any;
    getData: any;
    isValidateEditColumn: any;
    onUpdateLeftRightEditItems: any;
    isTableDouSelecOpen: any;
    coverReset: any;
    expandedKeysL?: any;
    expandedKeysR?: any;
    onMoveToLeft?: any;
    onMoveToRight?: any;
    onRightOrderChange?: any;
    filterable?: boolean;
}
export default class TableDoubleSelect extends Component<TableDoubleSelectProps> {
    constructor(props: any);
    shouldComponentUpdate(nextProps: any): boolean;
    render(): React.JSX.Element;
}
