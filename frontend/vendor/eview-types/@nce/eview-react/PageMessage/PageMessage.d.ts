import React, { Component, CSSProperties, ReactNode } from 'react';
export type PageMessageProps = {
    /**
     * 默认动态设置id，支持用户自定义id
     */
    id?: string;
    /**
     * 支持通过className自定义样式
     */
    className?: string;
    /**
     * 支持通过style自定义组件样式
     */
    style?: CSSProperties;
    /**
     * 支持通过iconClassName自定义图标样式
     */
    iconClassName?: string;
    /**
     * 支持通过text自定义文字提示内容
     */
    text?: string | ReactNode;
    /**
     * 支持通过icon设置组件图标，icon为url,组件图标默认符合AUI2.0规范，图片大小为16*16
     */
    icon?: any;
    /**
     * 默认:`default`<br>
     * 组件支持三种类型（type）:info、warn、default
     */
    type?: 'info' | 'warn' | 'default' | 'success' | 'error' | 'risk' | 'highRisk';
    /**
     * 是否显示外边框线
     */
    isShowBorder?: boolean;
};
export default class PageMessage extends Component<PageMessageProps> {
    constructor(props: PageMessageProps);
    static defaultProps: {
        type: string;
        isShowBorder: boolean;
        className: string;
    };
    render(): React.JSX.Element;
}
