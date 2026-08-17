import React, { Component } from 'react';
export type DivMessageProps = {
    /**
     * 默认支持动态设置id;同时支持用户自定义设置固定id
     */
    id?: string;
    /**
     * 支持通过className设置组件样式
     */
    className?: string;
    /**
     * 支持通过style设置组件样式
     */
    style?: React.CSSProperties;
    /**
     * 支持通过text设置组件文本内容
     */
    text?: string;
    /**
     * 支持通过icon设置组件图标，icon为url,组件图标默认符合AUI2.0规范，图片大小为16*16
     */
    icon?: string;
    /**
     * 支持通过iconClassName设置图标样式
     */
    iconClassName?: string;
    /**
     * 默认：`default`<br>
     * 支持通过type设置组件类型，包括[success，error，default]三种类型
     */
    type?: 'success' | 'error' | 'warn' | 'default';
    /**
     * 默认：`true`<br>
     * 支持通过display设置组件显示与隐藏
     */
    display?: boolean;
    /**
     * 自定义点击关闭按钮关闭DivMessage时的回调方法，不传则无回调
     * 例如：onClose={()=>alert('2')}
     */
    onClose?: (event?: any) => void;
    /**
     * 默认：`10000`<br>
     * 支持通过disposeTimeOut设置组件显示的时间.
     */
    disposeTimeOut?: number;
    /**
     * 默认：`true`<br>
     * 自动处理超时
     */
    enableDisposeTimeOut?: boolean;
    /**
     * 默认：`true`<br>
     * 支持通过closeIconFocus设置组件closeIcon图标是否聚焦
     */
    closeIconFocus?: boolean;
    /**
     * 默认：`true`<br>
     * 支持通过closeIconDisplay设置组件closeIcon图标显示与隐藏
     */
    closeIconDisplay?: boolean;
    /**
     * 默认：`true`<br>
     * lastFocus 设置成 false时，divmessage关闭时，焦点不会返回
     */
    lastfocus?: boolean;
    /**
     * 默认：`['auto','auto']`<br>
     * 自定义设置宽度
     */
    size?: string[];
    /**
     * 默认：`null`<br>
     * 设置提示信息的标题
     */
    title?: string;
    /**
     * 默认：`true`<br>
     * 设置是否显示 Icon
     */
    showIcon?: boolean;
    children?: React.ReactNode;
};
interface Istate {
    display: boolean;
}
export default class DivMessage extends Component<DivMessageProps, Istate> {
    static defaultProps: {
        type: string;
        display: boolean;
        disposeTimeOut: number;
        enableDisposeTimeOut: boolean;
        lastfocus: string;
        closeIconFocus: boolean;
        closeIconDisplay: boolean;
        size: string[];
        title: any;
        showIcon: boolean;
    };
    state: {
        display: boolean;
    };
    id: string;
    lastfocus: any;
    timerId: any;
    messageCloseIcon: HTMLSpanElement;
    componentWillReceiveProps(props: DivMessageProps): void;
    resetTimer(): void;
    componentDidMount(): void;
    componentDidUpdate(): void;
    componentWillUnmount(): void;
    handleClose: (event: React.MouseEvent<HTMLElement>) => void;
    handleKeyPress: (e: any) => void;
    render(): React.JSX.Element;
}
export {};
