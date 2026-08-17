import React, { Component } from 'react';
interface dataType {
    text?: string;
    tipData?: string;
    value?: any;
    icon?: string | React.ReactElement;
    iconActive?: string | React.ReactElement;
    label?: boolean;
    disabled?: boolean;
}
export type DropdownProps = {
    /**
     * 设置组件的id
     */
    id?: string;
    /**
     * 设置菜单项数据<br>
     * [{text: "Create",        * value: 1,       * icon:"image/create-default.png",      * disabled:true       *];
     */
    data?: dataType[];
    /**
     * 设置组件点击位置的文本
     */
    text?: string;
    /**
     * 通过传入路径设置图标按钮
     */
    iconUrl?: string | React.ReactElement;
    /**
     * 通过添加class的方式自定义组件下拉项中图标样式
     */
    iconClassName?: string;
    /**
     * 通过添加style的方式自定义组件下拉项中图标样式
     */
    iconStyle?: React.CSSProperties;
    /**
    /**
     * 通过添加style的方式，设置组件的样式,作用于组件的最外层div
     */
    style?: React.CSSProperties;
    /**
     * 通过添加class的方式，设置组件的样式,作用于组件的最外层div
     */
    className?: string;
    /**
     * 通过添加style的方式，设置组件下拉列表项的样式,作用于下拉的最外层div,可通过后代选择器控制下拉项的样式
     */
    itemStyle?: React.CSSProperties;
    /**
     * 通过添加class的方式，设置组件下拉列表项的样式,作用于下拉的最外层div,可通过后代选择器控制下拉项的样式
     */
    itemClassName?: string;
    /**
     * 通过添加class的方式自定义组件下拉项中图标样式
     */
    itemIconClassName?: string;
    /**
     * 通过添加style的方式自定义组件下拉项中图标样式
     */
    itemIconStyle?: React.CSSProperties;
    /**
     * 默认：`auto`<br>
     * 设置下拉菜单的位置
     */
    position?: 'left' | 'right' | 'auto';
    /**
     * 下拉按钮点击事件<br>
     * 签名：`function(display: bool) => void`<br>
     *  display 点击按钮之后，下拉菜单的展示状态，展示或关闭
     */
    onDropDown?: (display: boolean) => void;
    /**
     * 下拉列表项的点击事件<br>
     * 签名：`function(data: object) => void`<br>
     * data 被点击下拉项对应用户传入数据
     */
    onItemClick?: (item: dataType, e: any) => void;
    /**
     * 键盘按下事件
     */
    onKeyDown?: (e: any) => void;
    /**
     *  默认：`auto`<br>
     * 定义选择控件中弹出窗口的方向
     */
    popupDirection?: 'top' | 'bottom' | 'auto';
    /**
     *  默认：`9999`<br>
     * 设置zindex弹出窗口 .
     */
    zindex?: string;
    /**
     * 自动生成zindex，取页面中当前最高；谨慎使用，对性能有消耗
     */
    autoZindex?: boolean;
    /**
     *  弹出窗口关闭回调
     */
    onClosePopup?: any;
    /**
     *  默认：`false`<br>
     *值为true时，滚动条始终显示.
     */
    isScrollAlwaysDisplay?: boolean;
    /**
     *  默认：`8`<br>
     * 设置弹出窗口中显示的最大项目默认值为8
     */
    displayItems?: number;
    /**
     *  默认：`false`<br>
     * 禁用
     */
    disabled?: boolean;
    /**
     * 失焦事件
     */
    onBlur?: (e: React.MouseEvent<HTMLDivElement>) => void;
    /**
     * 点击事件
     */
    onClick?: () => void;
    /**
     *  默认：`click`<br>
     * 触发方式
     */
    trigger?: 'click' | 'hover';
    /**
     * 默认：`true`<br>
     *默认焦点第一项。
     */
    isAutoFirstFocus?: boolean;
    mountId?: string;
    children?: React.ReactNode;
    /**
     * 只显示风割线，不显示label
     */
    onlyShowDivider?: boolean;
    /**
     * 选中第几项
     */
    selectedIndex?: string | number;
    /**
     * 失焦隐藏的延迟时间
     */
    blurDelayShort?: boolean;
    /**
     *
     */
    hasBorder?: boolean;
};
type DropdownState = {
    value: string;
    displayPop: boolean;
    popPosition: string;
    position: string;
    displayTip: string;
};
export default class DropDown extends Component<DropdownProps, DropdownState> {
    static defaultProps: {
        position: string;
        popupDirection: string;
        zindex: string;
        disabled: boolean;
        trigger: string;
        isAutoFirstFocus: boolean;
        mountId: string;
    };
    id: string;
    currentFocus: any;
    firstFocus: any;
    lastFocus: any;
    allDisabled: boolean;
    dom: any;
    folding_arrow: any;
    blurTimer: any;
    pop: any;
    popOptions?: any;
    getValue: () => dataType;
    popupHolder: HTMLDivElement;
    antiShake: boolean;
    onBlur: boolean;
    constructor(props: any);
    /**
     * 计算下拉框的展开方向
     * @returns {boolean}
     */
    getPopPosition(): boolean;
    /**
     * 处理下拉面板展开或收起
     * @param e
     * @returns
     */
    handleFolding: (e: any) => void;
    isChild: (element: any) => boolean;
    handleOnBlur: (e: any) => void;
    handlePopMouseEnter: (e: any) => void;
    handlePopMouseLeave: (e: any) => void;
    handleListClick: (item: any, e: any) => void;
    handleTextClick: () => void;
    componentWillReceiveProps(nextProps: dataType): void;
    getPopClass(): string;
    collapseDropDown(): void;
    componentWillMount(): void;
    bindEvents: () => void;
    handleScroll: (e: any) => void;
    focusOnCurrentFocus: () => void;
    getFirstFocus: () => void;
    removeAndApplyKeyFocusStyle: (newFocus: React.KeyboardEvent<HTMLElement>) => void;
    /**
     *  捕获Enter事件，因为Enter事件很可能被Children处理了，不会冒泡出来
     * @param event
     */
    handleGlobalOnKeyDown: (event: any) => void;
    /**
     * Pop按键事件回调
     * @param event
     * @returns
     */
    handlePopOnKeyDown: (event: any) => void;
    handleOnKeyDown: (event: any) => void;
    componentDidMount(): void;
    componentWillUnmount(): void;
    componentDidUpdate(): void;
    /**
     * 设置弹窗的位置
     * @returns
     */
    setPopupPosition: () => void;
    render(): React.JSX.Element;
}
export {};
