import React, { Component } from 'react';
import { CheckboxProps, CheckboxState } from './types';
export default class Checkbox extends Component<CheckboxProps, CheckboxState> {
    static defaultProps: {
        checked: boolean;
        disabled: boolean;
        halfChecked: boolean;
        labelPosition: string;
        boxTabIndex: string;
        tipStyle: {};
    };
    tip: React.ReactInstance;
    checkboxDom: HTMLElement;
    id: string;
    state: {
        focus: boolean;
        halfChecked: boolean;
        checked: boolean;
        displayTip: boolean;
    };
    checkboxContainer: HTMLDivElement;
    getValue(): any;
    focus(): void;
    handleMouseUp: (e: any) => void;
    handleFocus: (e: any) => void;
    handleBlur: (e: any) => void;
    handleKeyPress: (e: any) => void;
    componentWillReceiveProps(nextProps: any): void;
    render(): React.JSX.Element;
}
