import React from 'react';
import Dialog from '../Dialog';
import { MessageDialogProps } from './types';
declare class MessageDialog extends Dialog {
    constructor(props: MessageDialogProps);
    closeIcon: any;
    msgDialogButtonDiv: any;
    static defaultProps: {
        modal: boolean;
        isOpen: boolean;
        closable: boolean;
        type: string;
        maxContHeight: number;
        closeOnEscape: boolean;
        zindex: number;
        mountId: string;
        minimizModalEnable: boolean;
        resizable: boolean;
        movable: boolean;
        minimizable: boolean;
        focusOnPanel: boolean;
        maskStyle: {};
        focusOnClose: boolean;
        destroyOnClose: boolean;
        customClose: boolean;
        size: any[];
        isAllowedExceed: boolean;
        animationOff: boolean;
        lastFocus: boolean;
        customMinimized: boolean;
        detailMessageTitle: React.JSX.Element;
        detailMessageShow: boolean;
        iconLocation: string;
        autoSetPosition: boolean;
    };
    componentDidMount(): void;
    componentDidUpdate(prevProps: any, prevState: any): void;
    onResize: () => void;
    componentWillUnmount(): void;
    static success(propss: any): {
        close(): void;
        open(): void;
    };
    handleKeyOnDetails: (e: any) => void;
    handleKeyOnClose: (e: any) => void;
    handleButtonKeyDown: (event: any) => void;
    handleOnClick: (e: any) => void;
    handleClick: (value: any, isChecked: any, event: any, additionalData: any) => void;
    componentWillReceiveProps(props: any): void;
    /**
     *焦点获取优先顺序逻辑
     * 1、如果弹窗有底部按钮区域，则按钮区域获取焦点，如果按鈕配置了focused屬性，則focused的按鈕聚焦，否則Cancel按鈕聚焦，沒有取消，則確認按鈕聚焦
     * （messageDialog只支持ok和cancel這兩種按鈕）
     * 2、X号获取焦点
     * 3、弹框获取焦点
     */
    focusOnButton: () => void;
    focus: () => void;
    handleOK: (event: any) => void;
    handleClose: (event: any) => void;
    handleCancel: () => void;
    handFocusBackToDialog: (event: any) => void;
    initButtons(): React.JSX.Element;
    handleDetailClick: (event: any) => void;
    initTitle(): any;
    render(): React.JSX.Element;
}
export default MessageDialog;
