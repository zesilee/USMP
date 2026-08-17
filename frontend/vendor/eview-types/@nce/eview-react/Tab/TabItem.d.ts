import React, { Component } from 'react';
export type TabItemProps = {
    /**
     * 设置TabItem的id
     */
    id?: string;
    /**
     *  每一项的样式类名。
     */
    className?: string;
    /**
     * tab的标题内容
     */
    title?: string;
    /**
     * 设置标签是否可关闭
     */
    closable?: boolean;
    /**
     * 设置Tab的灰化状态
     */
    disabled?: boolean;
    /**
     * Tab icon
     */
    icon?: string | React.ReactElement;
    /**
     * setEditing for tab
     */
    setEditing?: boolean;
    /**
     * set styling for tab header (font size, color etc..)
     */
    tabItemStyle?: React.CSSProperties;
    /**
     * 设置tab标题自定义的内容
     */
    titleExtraContent?: React.ReactElement;
    /**
     * 设置鼠标悬浮在item上的提示内容
     */
    itemTip?: string;
    index?: number;
    bar?: boolean;
    onClose?: (e: React.MouseEvent | React.KeyboardEvent, index: number, title: string) => void;
    onClick?: (e: React.MouseEvent | React.KeyboardEvent, index: number, title: string) => void;
    [other: string]: any;
};
export default class TabItem extends Component<TabItemProps> {
    static defaultProps: TabItemProps;
    id: string;
    handleClose: (e: React.MouseEvent) => void;
    handleClick: (e: React.MouseEvent) => void;
    handleKeyDown: (e: React.KeyboardEvent) => void;
    handleCloseKeyDown: (e: React.KeyboardEvent) => void;
    render(): React.JSX.Element;
}
