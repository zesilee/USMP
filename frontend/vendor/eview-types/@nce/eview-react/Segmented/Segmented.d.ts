import React, { Component } from 'react';
import { SegmentedProps, SegmentedState, ItemType } from './types';
export default class Segmented extends Component<SegmentedProps, SegmentedState> {
    id: any;
    static defaultProps: {
        type: string;
        labelPosition: string;
        required: boolean;
        isTipShow: boolean;
        disable: boolean;
    };
    getCheckedIndex: (data: ItemType[], value: string | number) => number;
    state: {
        checkedIndex: number;
    };
    segmented: HTMLDivElement;
    inkBar: HTMLDivElement;
    componentDidMount(): void;
    componentDidUpdate(prevProps: Readonly<SegmentedProps>, prevState: Readonly<SegmentedState>, snapshot?: any): void;
    componentWillReceiveProps(nextProps: any): void;
    handleChange: (index: number, e: any) => void;
    getValue: () => string | number;
    getClassName: (index: any, item: any) => string;
    handleKeyDown: (e: any, size: any) => void;
    setInBarPosition: () => void;
    render(): React.JSX.Element;
}
