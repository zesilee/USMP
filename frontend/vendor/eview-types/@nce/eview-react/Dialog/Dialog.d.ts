import React, { Component } from 'react';
import { DialogProps, DialogState } from './types';
export default class Dialog extends Component<DialogProps, DialogState> {
    panelWidth: number;
    orientation: boolean;
    position: any;
    isSizeChange: boolean;
    observer: ResizeObserver | null;
    protected closeBtnRef: any;
    /** Dialog弹窗布局改成fixed，所以不再需要scrollTop和 scrollLeft*/
    protected scrollTop: any;
    protected scrollLeft: any;
    protected top: any;
    protected lastFocus: any;
    protected index: any;
    protected elementArray: any;
    protected dialogDom: any;
    protected button: any;
    protected cancelBtn: any;
    protected panelHeight: any;
    protected left: any;
    protected dragMoveEndTimer: any;
    protected dialogButtonDiv: any;
    protected startX: any;
    protected startY: any;
    protected startWidth: any;
    protected startHeight: any;
    protected startTop: any;
    protected startLeft: any;
    protected bodyContent: any;
    protected id: any;
    protected resize: any;
    protected dialogOver: any;
    protected prevLeftVal: any;
    constructor(props: any);
    static defaultProps: {
        modal: boolean;
        minimizModalEnable: boolean;
        isOpen: boolean;
        closable: boolean;
        resizable: boolean;
        movable: boolean;
        minimizable: boolean;
        closeOnEscape: boolean;
        focusOnPanel: boolean;
        zindex: number;
        mountId: string;
        maskStyle: {};
        focusOnClose: boolean;
        destroyOnClose: boolean;
        customClose: boolean;
        size: string[];
        isAllowedExceed: boolean;
        animationOff: boolean;
        lastFocus: boolean;
        customMinimized: boolean;
        autoSetPosition: boolean;
    };
    handFocusBackToDialog: (event?: any) => void;
    handleKeyOnClose: (e: any) => void;
    handleKeyCustomIcon: (e: any, list: any) => void;
    getElement: () => any;
    /**
     *焦点获取优先顺序逻辑
     * 1、如果弹窗有底部按钮区域，则按钮区域获取焦点，如果按鈕配置了focused屬性，則focused的按鈕聚焦，否則最後一個按鈕聚焦
     * 2、X号获取焦点
     * 3、弹框获取焦点
     */
    lastFocusFunc: (closeIcon: any) => void;
    getAllElement: () => void;
    focus(): void;
    keyDownOnDialogPanel: (event: any) => void;
    componentWillUpdate(nextProps: any): void;
    componentWillMount(): void;
    componentDidUpdate(prevProps: any, prevState: any): void;
    initPanelStyle: ({ position, size, zIndex, style, title, }: {
        position?: Array<number | null>;
        size?: Array<number | string>;
        zIndex?: number;
        style?: React.CSSProperties;
        title?: string | React.ReactNode;
    }) => React.CSSProperties;
    centerDoms: (dialogRef: any) => {
        left: number;
        top: number;
    };
    panelStyleRefresh: (isHeightChange: boolean) => void;
    modifyIsSizeChange: () => void;
    componentDidMount(): void;
    updateFocusOnDialogButtons(): void;
    /**
     * MessageDialog继承了Dialog
     * Dialog组件中首次加载设置默认焦点时是根据button按钮是否有primary属性或者默认给最后一个按钮设置焦点
     * 这与MessageDialog设置焦点的逻辑（可以设置button的focused值，focused为true则获取焦点）冲突
     * 所以重新写一个函数用于MessageDialog聚焦
     */
    handleMessageDialogFocus: () => void;
    setPosition(): void;
    isMoved(): boolean;
    componentWillUnmount(): void;
    componentWillReceiveProps(props: any): void;
    getStyle: () => React.CSSProperties;
    centerDom: (dire: any) => void;
    getUrl(): React.JSX.Element;
    handleClose: (event: any) => void;
    handleCustomIcon: (event: any, list: any) => void;
    handleMinimize: (e: any) => void;
    handleMaximize: (e: any) => void;
    handleButtonKeyDown: (event: any) => void;
    mouseDown: (e: any) => void;
    /**
     * 拖拽移动时
     * @param e
     */
    mouseMove: (e: any) => void;
    mouseUp: (e: any) => void;
    getDialogPosition: () => void;
    /**
     * 设置drag过程中的位置，增加判断，不能超出边界
     * @param e
     */
    getDragposition: (e: any) => void;
    getDefaultStyle: (obj: any, attribute: any) => any;
    initButtonArea(): React.JSX.Element;
    handleKeyOnMinimize: (e: any) => void;
    handleKeyOnMaximize: (e: any) => void;
    initTitleBarArea(): React.JSX.Element;
    resetScroll: () => void;
    initResize: (e: any) => void;
    doResize: (e: any) => void;
    stopDrag: (e: any) => void;
    handleChangingMinimize: (minimize: boolean) => () => void;
    render(): React.JSX.Element;
}
