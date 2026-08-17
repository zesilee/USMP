import React, { Component } from 'react';
import { RadioProps, RadioState } from './types';
export default class Radio extends Component<RadioProps, RadioState> {
    static defaultProps: {
        checked: boolean;
        disabled: boolean;
        labelPosition: string;
        isControlled: boolean;
    };
    tip: any;
    radioDom: HTMLElement;
    id: any;
    state: RadioState;
    radioContainer: HTMLDivElement;
    componentWillReceiveProps(nextProps: any): void;
    componentDidUpdate(): void;
    getValue(): any;
    handleClick: (e?: React.MouseEvent<HTMLElement>) => void;
    handleFocus: (e: any) => void;
    handleBlur: (e: any) => void;
    handleKeyPress: (e: any) => void;
    focus(): void;
    render(): React.JSX.Element;
}
