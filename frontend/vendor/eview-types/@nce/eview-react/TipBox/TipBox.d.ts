import React, { Component } from 'react';
import { Trigger, Direction } from './constant';
export type TipBoxProps = {
    /**
     * 默认支持动态设置id;支持用户自定义设置固定id
     */
    id?: string;
    /**
     * 支持通过className设置样式
     */
    className?: string;
    /**
     * 支持通过style设置样式
     */
    style?: React.CSSProperties;
    /**
     * 支持通过title设置标题
     */
    title?: string;
    /**
     * 支持通过text设置内容
     */
    content?: any;
    /**
     * 默认：`bottom`<br>
     * 支持通过arrowDirection设置箭头方向：上 下 左 右
     * left'| 'right' |'top'| 'bottom'| 'none'
     */
    arrowDirection?: any;
    /**
     * 支持通过position设置组件定位position = [top,left]，如[20,30]
     */
    position?: [number, number];
    /**
     * 原有的position只支持四方向，通过direction属性可以支持12方向
     */
    direction?: Direction;
    /**
     * 默认：`true`<br>
     * 支持通过display设置组件显示与隐藏,有children的场景不支持
     */
    display?: boolean;
    displayMode?: string;
    /**
     * 默认：`false`<br>
     * Tipbox是否可以手动关闭
     */
    isClosable?: boolean;
    /**
     * 默认：`true`<br>
     * 鼠标离开的时候，是否自动关闭
     */
    isMouseLeaveClose?: boolean;
    /**
     * 默认：`false`<br>
     * 是否是出错提示框
     */
    isErrorTip?: boolean;
    /**
     * 默认：`error`<br>
     * 出错提示框的标题
     */
    errorTitle?: string;
    /**
     * 默认：`error`<br>
     * 出错提示框的内容(同时情况)
     */
    errorContent?: React.ReactNode | string;
    /**
     * 默认：`0`<br>
     * 应该隐藏提示，如果用户给出超过 0 毫秒，其他明智就不会隐藏。
     */
    disposeTimeOut?: number;
    errorInputClassName?: string;
    onClose?: (event: React.FocusEvent<HTMLInputElement>) => void;
    /**
     * Tipbox自动关闭回调
     */
    onDispose?: () => void;
    onScroll?: any;
    /**
     * Set the title style
     */
    titleStyle?: React.CSSProperties;
    titleClassName?: string;
    onMouseLeave?: (event: React.MouseEvent) => void;
    onMouseEnter?: (event: React.MouseEvent) => void;
    animationTime?: number;
    /**
     * 触发方法
     */
    trigger?: Trigger | Trigger[];
    children?: React.ReactNode;
    /**
     * 自动生成zindex，取页面中当前最高；谨慎使用，对性能有消耗
     */
    autoZindex?: boolean;
    type?: 'normal' | 'simple';
    /**
   * 触发tipbox显示的dom元素
   */
    triggerNode?: HTMLElement;
};
interface TipBoxState {
    display: boolean;
    isOpen: boolean;
    isPageHidden: boolean;
}
export default class TipBox extends Component<TipBoxProps, TipBoxState> {
    timeoutUpdate: number | null;
    timeoutMount: number | null;
    id: string;
    baseRef: any;
    tipbox: HTMLElement | null;
    delayTimer: number | null;
    isInTipbox: boolean;
    direction: Direction | null;
    coordinate: {
        arrowDirection: 'left' | 'right' | 'top' | 'bottom' | 'none';
        placement: string;
    } | null;
    tipboxArrow: HTMLDivElement;
    orientation: boolean;
    observer: ResizeObserver | null;
    updateLock: boolean;
    lastHiddenTime: number | null;
    constructor(props: TipBoxProps);
    static defaultProps: {
        arrowDirection: string;
        direction: string;
        disposeTimeOut: number;
        isClosable: boolean;
        isErrorTip: boolean;
        errorTitle: string;
        errorContent: string;
        className: string;
        isMouseLeaveClose: boolean;
        trigger: string;
        animationTime: number;
        type: string;
    };
    componentWillReceiveProps(nextProps: TipBoxProps): void;
    handleClickOutSide: React.MouseEventHandler<HTMLElement>;
    handleScroll: (e: any) => void;
    clearTimeoutIfSet(timeoutId: number | null): void;
    autoDispose(): number | null;
    getChildrenDom(): Element;
    componentDidMount(): void;
    componentWillUpdate(): void;
    componentDidUpdate(): void;
    componentWillUnmount(): void;
    isMouseInElement: (e: MouseEvent) => boolean;
    handleMouseEnter: (event: any) => void;
    handleMouseLeave: (event: any) => void;
    handleColseKeyDown: (e: any) => void;
    handleVisibilityChange: () => void;
    handleClose: (e: any) => void;
    /**
     * 获取tipbox的箭头className
     * @returns
     */
    getclassName(): string;
    modifyClassName(baseDom: HTMLElement): void;
    adjustArrowPosition(baseDom: HTMLElement, tipboxArrow: HTMLElement): void;
    /**
     * 根据参照元素设置Tipbox的位置
     * @param baseDom
     * @returns
     */
    setPositon(baseDom: any, forceDirection?: any): void;
    clearDisposeTimeOut: () => void;
    togglePopup: (nextOpen: boolean) => void;
    delaySetPopupVisible: (nextOpen: boolean, delayS: number) => void;
    getChildren(children: any, trigger: any): React.DetailedReactHTMLElement<React.RefAttributes<HTMLElement> & React.HTMLAttributes<HTMLElement>, HTMLElement>;
    isReactElement(value: unknown): value is React.ReactElement;
    render(): React.JSX.Element;
}
export {};
