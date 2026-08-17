import React, { Component } from 'react';
export type HelpTipProps = {
    /**
     * id:定制组件 dom 元素 id，传入该值可覆盖组件自动生成的 id
     */
    id?: string;
    /**
     * 默认：`[1rem,1rem]`<br>
     * 定制图标的大小，不传则使用默认大小
     */
    iconSize?: string[];
    className?: string;
    /**
     * 支持通过 style 设置 tip 提示框的样式
     */
    tipStyle?: React.CSSProperties;
    /**
     * 默认：`[0,0]`<br>
     * 设置图标的位置，图标本身的 display 属性为 relative，
     * position[0]设置 top，position[1]设置 left
     */
    position?: number | string[];
    /**
     * 设置提示框里的内容，默认值为空,提示框里的内容支持任意 react 元素
     */
    tipContent?: any;
    /**
     * 默认：`left`<br>
     * 支持通过 arrowDirection 设置箭头方向：上 下 左 右
     */
    arrowDirection?: 'left' | 'right' | 'top' | 'bottom';
};
type HelpTipState = {
    display: boolean;
};
export default class HelpTip extends Component<HelpTipProps, HelpTipState> {
    tip: any;
    iconRef: HTMLElement;
    id: string;
    static defaultProps: {
        position: number[];
        iconSize: string[];
        tipStyle: {};
        tipContent: string;
        arrowDirection: string;
    };
    constructor(props: any);
    componentDidUpdate(): void;
    handleMouseOver: () => void;
    handleMouseLeave: (event: any) => void;
    handleKeyDown: (event: React.KeyboardEvent) => void;
    handleKeyUp: (event: React.KeyboardEvent) => void;
    setHelpTipContainerSize(size: any): string;
    render(): React.JSX.Element;
}
export {};
