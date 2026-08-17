import React, { Component, CSSProperties } from 'react';
interface dataType {
    text?: string;
    tipData?: string;
    value?: string;
    icon?: string | React.ReactElement;
    iconActive?: string | React.ReactElement;
    label?: boolean;
    disabled?: boolean;
}
export type PopupProps = {
    /**
     * 设置组件id
     */
    id?: string;
    /**
     * 通过添加class的方式自定义组件样式，作用于组件最外层div
     */
    className?: string;
    /**
     * 通过添加style的方式自定义组件样式，作用于组件最外层div
     */
    style?: any;
    /**
     * 通过添加style的方式自定义组件下拉项样式
     */
    itemStyle?: React.CSSProperties;
    /**
     * 通过添加class的方式自定义组件下拉项样式
     */
    itemClassName?: string;
    /**
     * 通过添加class的方式自定义组件下拉项中图标样式
     */
    iconClassName?: string;
    /**
     * 通过添加style的方式自定义组件下拉项中图标样式
     */
    iconStyle?: React.CSSProperties;
    /**
     * 默认：`false`<br>
     * 设置组件是否显示
     */
    display?: boolean;
    /**
     * 设置传入数据
     * <br>[{
     * <br>text: "Create",        *
     * <br>value: 1,       *
     * <br>icon:"image/create-default.png",      *
     * <br>iconActive:"image/create-selected.png",      * <br>disabled:true       * <br>}, ...]
     */
    data?: any;
    /**
     * 通过value设置组件的选中项
     */
    value?: any;
    /**
     * 设置组件的onClick事件<br>
     * 签名：`(item: dataType, e: any) => void;`<br>
     * (object) data 被点击下拉项对应用户传入数据
     */
    onItemClick?: (item: dataType, e: any) => void;
    /**
     * Triggers an event when mousedown  occurs on component
     * {Object} event Object:{event, index}
     */
    onMouseDown?: (e: any, index: number) => void;
    /**
     *如果内容要在滚动上动态加载，而不是在一次镜头中预加载所有项目，则将选项传递到菜单。
     *这将确保弹出窗口更快，即使要显示大量项目，如在搜索组件中，搜索结果为1000个项目
     *'总记录：1000条，
     * onLoadRecords（开始索引），
     *`
     */
    lazyLoadOptions?: any;
    /**
     * 默认：`auto`<br>
     *定义弹出窗口的方向。这将导致弹出窗口的位置与其高度相同，以达到默认预期位置的顶部。
     *这仅在理想情况下选择组件弹出的情况下才有意义。
     */
    popupDirection?: 'top' | 'bottom' | 'auto' | string;
    /**
     * 默认：`8`<br>
     *设置弹出窗口最多显示的项目默认8个
     */
    displayItems?: number;
    /**
     * 关闭Popup的回调函数
     */
    onClosePopup?: () => void;
    onOpenPopup?: any;
    /**
     * 默认：`true`<br>
     *值为true，滚动条始终显示。
     */
    isScrollAlwaysDisplay?: boolean;
    /**
     * 默认：`true`<br>
     *默认焦点第一项。
     */
    isAutoFocus?: boolean;
    /**
     * 默认：`true`<br>
     *设置弹出窗口全键宽。
     */
    isPopupKeyboard?: boolean;
    handleListClick?: (item: dataType, e: any) => void;
    /**
     * 键盘按下后的回调函数
     */
    onKeyDown?: (e: React.KeyboardEvent<HTMLDivElement>) => void;
    /**
     * 键盘按下Esc后的回调函数
     */
    onEscCallBack?: (item: dataType) => void;
    enablHorzScroll?: any;
    /**
     * 鼠标滚动的回调函数
     */
    onScroll?: any;
    /**
     * 鼠标移出Popup的回调函数
     */
    onMouseLeave?: any;
    /**
     * 鼠标划入Popup的回调函数
     */
    onMouseEnter?: any;
    /**
     * 是否支持虚拟滚动,所有基于Popup的组件都可以实现虚拟滚动
     */
    virtualScroll?: boolean;
    startIndex?: number;
    optionStyle?: CSSProperties;
    hasAnimation?: boolean;
    /**
     * 只显示风割线，不显示label
     */
    onlyShowDivider?: boolean;
    /**
    * 是否展示加载效果
    */
    isLoading?: boolean;
    /**
    * 是否在lazyLoadOptions更改时设置顶部定位为0
    */
    changeScroll?: boolean;
};
type PopupState = {
    display: boolean;
    ActiveIndex: any;
    curIndex: number;
    isAutoFocus: boolean;
    menuTopPosition: number;
    data: dataType[];
    focusIndex: any;
    cycleIndex: number;
    flag: boolean;
    startIndex: number;
    isLoading: boolean;
};
export default class Popup extends Component<PopupProps, PopupState> {
    id: any;
    selectedIdx: number;
    scrollBufferLen: number;
    scrollTop: number;
    lastFocus: any;
    element: any;
    lazyContainer: HTMLDivElement;
    totalRecordsConatiner: React.ReactNode;
    static defaultProps: {
        display: boolean;
        isAutoFocus: boolean;
        data: any[];
        popupDirection: string;
        popupWidth: number;
        isScrollAlwaysDisplay: boolean;
        isPopupKeyboard: boolean;
    };
    static contextType: React.Context<import("react-intl").IntlShape>;
    scrollOuter: HTMLDivElement;
    scrollInner: HTMLDivElement;
    deltaY: any;
    scroll: any;
    constructor(props: PopupProps, context: any);
    componentWillReceiveProps(nextProps: PopupProps): void;
    handleParentScroll: (e: any) => void;
    componentDidMount(): void;
    componentDidUpdate(prevProps: PopupProps, prevState: PopupState): void;
    componentWillUnmount(): void;
    handleUpAndDownKeying: (event: any) => void;
    checkInitUnDisableditems(): void;
    isOptionAllDisable: (option: any) => boolean;
    /**
     * 处理按键的上下移动
     * @param event
     * @returns
     */
    handleUpAndDownKey: (event: any) => void;
    handleOptionMouseDown: (index: number) => (e: React.MouseEvent) => void;
    handleOptionClick: (item: dataType) => (e: React.MouseEvent<HTMLDivElement>) => void;
    handleScroll: (event: any) => void;
    handleOptionsScroll: (e: any) => boolean;
    /**
     * wheel的时候，只记录deltaY，scroll的时候再处理
     * @param e
     * @returns
     */
    handleOptionsWheel: (e: any) => boolean;
    getVirtualScrollContent: () => React.JSX.Element;
    getLazyContent: () => React.JSX.Element;
    /**
     * 生成子项
     * @returns
     */
    generateItems: () => any;
    getContent(): React.ReactNode;
    handlePreventScroll: (event: any) => void;
    render(): React.JSX.Element;
}
export {};
