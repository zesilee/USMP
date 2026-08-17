import React, { Component } from 'react';
import { ButtonProps, ButtonState } from './types';
export default class Button extends Component<ButtonProps, ButtonState> {
    private isShowTip;
    private isContentShouldUpdate;
    private tip;
    private id;
    dom: any;
    private buttonText;
    constructor(props: any);
    static defaultProps: {
        text: string;
        status: string;
        type: string;
        disabled: boolean;
        focused: boolean;
        size: string;
        tipType: string;
        leftIconProps: {
            leftHoverIcon: any;
            leftDisabledIcon: any;
            leftIconClass: any;
            leftIconDisabledClass: any;
        };
        rightIconProps: {
            rightHoverIcon: any;
            rightDisabledIcon: any;
            rightIconClass: any;
            rightIconDisabledClass: any;
        };
        disableBlur: boolean;
    };
    private getTextPadding;
    changeStyle: (style: any) => void;
    componentDidMount(): void;
    componentWillReceiveProps(nextProps: Readonly<ButtonProps>): void;
    private initCallBack;
    componentDidUpdate(): void;
    private getClass;
    private handleKeyPress;
    private handleClick;
    private handleOnFocus;
    private handleMouseOver;
    private handleMouseOut;
    handleBlur: () => void;
    render(): React.JSX.Element;
}
