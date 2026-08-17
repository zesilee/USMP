import React, { Component } from 'react';
import { RadioGroupProps, RadioGroupState } from './types';
export default class RadioGroup extends Component<RadioGroupProps, RadioGroupState> {
    static defaultProps: {
        disabled: boolean;
        required: boolean;
        labelPosition: string;
        isControlled: boolean;
        type: string;
        hintType: string;
    };
    state: {
        hasValue: boolean;
        checkedIndex: any;
        showRequiredTip: boolean;
    };
    id: any;
    requiredFlag: any;
    radioContent: any;
    tip: any;
    getCheckedIndex(data: any, value: any): any;
    handleScroll: () => void;
    handleBlur: () => void;
    handleMouseOut: (e: any) => void;
    handleChange: (index: any) => (val: any, e: any) => void;
    focus(): void;
    getValue(): any;
    validate(): boolean;
    componentWillReceiveProps(nextProps: any): void;
    isInArray: (a: any, arrayObj: any) => boolean;
    rowRadioButton(radiogroups: any, radio: any, rowSpacing: any): any;
    componentDidUpdate(): void;
    render(): React.JSX.Element;
}
