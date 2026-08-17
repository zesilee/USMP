import React, { Component, CSSProperties, ReactNode } from 'react';
type HtmlLabelProps = React.DetailedHTMLProps<React.LabelHTMLAttributes<HTMLLabelElement>, HTMLLabelElement>;
export interface LabelFieldProps extends HtmlLabelProps {
    /**
     * 通过style的方式自定义样式
     */
    style?: CSSProperties;
    /**
     * 通过class的方式自定义样式
     */
    className?: string;
    /**
     *  支持通过text设置组件文本内容
     */
    text?: string | number;
    /**
     * 支持设置组件的子标签
     */
    children?: ReactNode;
    /**
     * 设置组件的点击事件
     * @param event 原生点击事件
     */
    onClick?: (event: React.MouseEvent<HTMLLabelElement, MouseEvent>) => void;
    /**
     *  组件默认设置动态id，且组件支持用户自定义id
     */
    id?: string;
    /**
     * 设置是否是必填项，为true时，文字前面加星号
     */
    required?: boolean;
    /**
     * 设置是否自定义宽度，为true时，宽度固定，支持单行省略，中英文切换时自动切换宽度？
     */
    enableFixWidth?: 'small' | 'middle' | 'large' | 'none';
    ref?: any;
}
export default class LabelField extends Component<LabelFieldProps> {
    static defaultProps: {
        enableFixWidth: string;
    };
    constructor(props: any);
    getClassName: (className: any) => string;
    render(): React.JSX.Element;
}
export {};
