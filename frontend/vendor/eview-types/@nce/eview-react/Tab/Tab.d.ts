import React, { Component } from 'react';
import TabItem from './TabItem';
import TabContent from './TabContent';
type typeT = 'main' | 'sub' | 'split';
type positionType = 'top' | 'bottom' | 'left' | 'right';
export type TabProps = {
    /**
     * 设置Tab的id
     */
    id?: string;
    /**
     * 设置选中的index,此index是传入数据数组下标
     */
    selectedIndex?: number;
    /**
     * 通过自定义class来自定义Tab的整体样式
     * 例如：margin、padding
     */
    className?: string;
    /**
     * 默认：`false`<br>
     * 禁用所有Tab
     */
    disabled?: boolean;
    /**
     * 默认：`false`<br>
     * 用hover切换tab
     */
    hover?: boolean;
    /**
     * 通过自定义style来自定义Tab的整体样式
     * 例如：margin、padding
     */
    style?: React.CSSProperties;
    /**
     * 默认：`main`<br>
     * 设置页签的类型，可选`main`,`sub`。main代表是一级页签，也是默认类型，sub表示二级页签
     */
    type?: typeT;
    /**
     * 默认：`top`<br>
     * 可选：`top`,`bottom`,`left`,`right`。
     * Set the position/alignment of the tabs，default indicates the top alignment of the tab and it's content
     */
    position?: positionType;
    /**
     * 默认：`false`<br>
     * 设置是否在页签激活时动态加载内容，默认首次渲染时全部加载
     */
    lazyLoad?: boolean;
    /**
     * 设置Tab页签的切换回调事件<br>
     * 签名：`function(index: number, title: string, event: object) => void`<br>
     * index 被选中的页签在用户传入数组中的index<br>
     * title: tabName<br>
     * {object} event event事件
     */
    onClick?: (index: number, title: string, e: React.MouseEvent | React.KeyboardEvent) => void;
    /**
     * 设置Tab是否可关闭<br>
     *  签名：`function(index: number, event: object, title: string) => void`<br>
     * {number} index 被选中的页签在用户传入数组中的index<br>
     *  {object} event event事件<br>
     * title: tabName
     */
    onClose?: (index: number, e: React.MouseEvent | React.KeyboardEvent, title: string) => void;
    /**
     * 默认：`true`<br>
     * 默认情况下，Tab 项是可拖动的。将此属性设置为 false 以使 tab-items 不可拖动示例：draggable = {false}
     */
    draggable?: boolean;
    /**
     * 默认：`true`<br>
     * set tab items AutoClose or not
     */
    isAutoClose?: boolean;
    /**
     * 默认：`false`<br>
     * 设置是否显示关闭选中，其他，所有等按钮，默认不显示
     */
    isShowCloseBtns?: boolean;
    /**
     * 默认：`false`<br>
     * 是否通过其他元素关闭TabItem,默认为false
     */
    isCloseByTabIds?: boolean;
    /**
     * Tab标签页关闭前手动触发<br>
     * 签名：`function(tabIds: array) => void`<br>
     * tabIds: 需要确认关闭的页签 index 组成的数组，前提是 isCloseByTabIds 设为 true
     */
    onBeforeClose?: (tabIds: []) => void;
    /**
     * 通过控制tabIds关闭后的回调<br>
     * 签名：`function(tabIds: array) => void`<br>
     * tabIds: 关闭的页签 index 组成的数组
     */
    onCloseByTabIds?: (tabIds: [] | '', buttonIdentify: number) => void;
    /**
     * 返回title拖拽后新排序的节点信息
     */
    ondragEnd?: (nodeAfterDrag: {
        id: string;
        title: string;
    }[], e: React.MouseEvent) => void;
    /**
     * 通过自定义class来自定义tabContent的样式
     */
    tabContentClassName?: string;
    /**
     * 通过自定义style来自定义tabContent的样式
     */
    tabContentStyle?: React.CSSProperties;
    /**
     * lazyLoad="false"状态,切换页签内容区是否更新，默认false不更新
     */
    isUpdateContent?: boolean;
    children?: any;
    /**
     * 设置收纳项是否全部标题，onlyHideNone默认为false显示全部标题，如果设置为true，则显示隐藏的部分标题
     */
    onlyHideNone?: boolean;
    /**
     * 通过自定义headStyle来自定义tab标题区域样式,上下方向不建议通过它设置宽度
     */
    headStyle?: React.CSSProperties;
    /**
     * 监听tab最外层div宽度是否变化，从而重新渲染tab
     */
    observerWidthChange?: boolean;
    /**
     * 监听tabItem最外层div宽度是否变化，从而重新渲染tab
     */
    observerTabItemChange?: boolean;
    [restprop: string]: any;
};
type TabState = {
    selectedIndex: number;
    itemStatus: any;
    popDisplay: string;
    widthArr: any[] | null;
    dragIndex: number | null;
    dragActive: boolean;
    dragArr: any;
    topDisable: boolean;
    botmDisable: boolean;
    showArr: any;
    showLRArr: any;
};
type defaultProps = Pick<TabProps, 'selectedIndex' | 'type' | 'lazyLoad' | 'position' | 'draggable' | 'isAutoClose' | 'isShowCloseBtns' | 'isCloseByTabIds' | 'isUpdateContent' | 'onlyHideNone'>;
declare class Tab extends Component<TabProps, TabState> {
    static contextType: React.Context<import("react-intl").IntlShape>;
    static defaultProps: defaultProps;
    private currentFocus;
    private buttonIdentify;
    private tabsCount;
    private displayMore;
    private firstUpdate;
    private id;
    private pop;
    private tabHeader;
    private tabWidth;
    private showTitleRef;
    tabContent: TabContent;
    tabItemRef: TabItem;
    static TabItem: typeof TabItem;
    formatMessage: any;
    inkBar: HTMLDivElement;
    resizeObserver: any;
    constructor(props: TabProps, context: any);
    state: TabState;
    handleScroll: (e: any) => void;
    getSelectIndex(selectedIndex: any, children: any): any;
    initDragArr(param: any): any[];
    getItemStatus(param: any, isShow: any): any[];
    handleUpAndDownKey: (event: any) => void;
    handleClick: (e: any, index: any, title: any) => void;
    handleClose: (e: any, index: any, title: any) => void;
    handleFolding: (e: any) => void;
    handleClearPopUp: () => void;
    isChild: (element: any) => boolean;
    handleBlur: (e: any) => void;
    handleParentResize: () => void;
    updateTabWidth: () => void;
    getShowTitleArr(widthArr: any, selectedIndex: any, width?: number | undefined): any[];
    updateTopBottom(): void;
    updateLeftRight(): void;
    getTabTitle(param?: string): any;
    afterCloseFocus: () => void;
    handleRealClose(e: any, index: any, title: any): void;
    beforeCloseItems(tabIds: any, callback: any): void;
    closeItemByTabIds(TabIds: any): void;
    closeTabByBtn(btnIndex: any): void;
    getCloseBtns(): React.JSX.Element;
    handleDragStart(index: number): (e: any) => void;
    handleDragEnd: (e: any) => void;
    handleDragEnter: (index: any) => (e: any) => void;
    handleUp: (e: any) => void;
    handleDown: (e: any) => void;
    componentDidMount(): void;
    setInBarPosition: () => void;
    componentWillReceiveProps(nextProps: any): void;
    componentDidUpdate(): void;
    componentWillUnmount(): void;
    render(): React.JSX.Element;
}
export default Tab;
