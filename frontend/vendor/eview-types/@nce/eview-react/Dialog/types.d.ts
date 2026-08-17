import React from 'react';
interface Boundary {
    top?: number | string;
    right?: number | string;
    bottom?: number | string;
    left?: number | string;
}
export type DialogProps = {
    /**
     *定制组件dom元素id，传入该值可覆盖组件自动生成的id
     * 例如：
     * id='DialogId'
     */
    id?: string;
    /**
     * 定制组件dom元素className，传入该值可追加组件自动生成的className，从而自定义组件样式
     * 例如：
     * className='DialogClassName'
     */
    className?: string;
    /**
     * 定制弹出框标题，不传则标题为空
     * 例如：
     * title='Dialog'
     */
    title?: any;
    /**
     * 定制弹出框自定义图标,不传则不显示
     * 例如：
     * {iconUrl?: 'image/huawei.png', customIconClick?: function() {alert('1111')}, tip?: '北京'},
     * {iconUrl?: 'image/loading.gif', customIconClick?: function() {alert('2222')}, tip?: '上海'},
     * {iconUrl?: 'image/header.png', customIconClick?: function() {alert('3333')}, tip?: '广东'},
     */
    customIcons?: any;
    /**
     *自定义弹出框传入第三方页面url，嵌入第三方页面
     * 例如：
     * url='test1.html'
     */
    url?: string;
    /**
     *自定义弹出框显示位置，两个数字的数组[x,y]，x标识左边距，y标识上边距，不传则使用默认位置
     * <br/>例如：position：[200,200]
     * <br/>只设置左边距，不设置上边距时，例如：[300，null]
     * <br/>只设置上边距，不设置左边距时，例如：[null，300]
     */
    position?: any;
    /**
     *自定义弹出框大小，两个数字的数组[x,y]，x标识宽度，y标识高度，不传则使用默认大小
     * <br/>例如：size=[300,300]
     * <br/>只设置宽度，不设置高度时，例如：[300，null]
     * <br/>只设置高度，不设置宽度时，例如：[null，300]
     */
    size?: any;
    /**
     * 默认：`false`<br>
     *弹出框最小化后模态是否显示，不传，则默认为false
     * 例如：
     * minimizModalEnable={false}
     */
    minimizModalEnable?: boolean;
    /**
     * 默认：`true`<br>
     *自定义弹出框是否显示关闭按钮，默认值为true（可关闭）
     * 例如：
     * closable={false}
     */
    closable?: boolean;
    /**
     * 默认：`false`<br>
     *自定义弹出框是否显示，默认值为false（关闭）
     * 例如：
     * isOpen={true}
     */
    isOpen?: boolean;
    /**
     * 默认：`true`<br>
     *自定义弹出框是否模态，不传，则默认为true
     * 例如：
     * modal={false}
     */
    modal?: boolean;
    /**
     * 默认：`false`<br>
     *To control the window resize property.
     * Default：
     * resizable={false}
     */
    resizable?: boolean;
    /**
     * 默认：`true`<br>
     *To control the window resize property.
     * Default：
     * movable={false}
     */
    movable?: boolean;
    /**
     *自定义弹出框样式
     * 例如：
     * style={{color?:'red'}}
     */
    style?: React.CSSProperties;
    /**
     *自定义弹出框内容区样式
     * 例如：
     * contentStyle={{color?:'red'}}
     */
    contentStyle?: object;
    /**
     *自定义弹出框按钮对齐样式,前提是传入了button
     * 例如：
     * buttonStyle={{text-align?:'center'}}
     */
    buttonStyle?: object;
    /**
     *自定义弹出框按钮区按钮
     * 例如：
     * buttons= [{text?:'确定',onClick?:()=>{alert('2')}}]
     */
    buttons?: any;
    /**
     *自定义弹出框传入页面内容，支持React标签以及HTML原生标签，有两种传入方式
     * 例如：
     * <br/>1, children={&lt;div&gt;这是传入的内容&lt;/div&gt;}
     * <br/>2, &lt;Dialog&gt;这是传入的内容&lt;/Dialog&gt;
     */
    children?: any;
    /**
     *自定义点击关闭按钮关闭弹出框时的回调方法，不传则无回调<br>
     * 签名：`function(event: object) => void`<br>
     *  event 原生dom事件
     */
    onClose?: (event: React.MouseEvent | React.KeyboardEvent) => void;
    /**
     *When the user set the resizable property is true then user resize the dialog this callback will be triggerd, the parameter is dialog size object .
     */
    onResize?: (obj: object) => void;
    /**
     * 键盘事件
     */
    onKeyDown?: any;
    /**
     * 默认：`true`<br>
     *关闭键盘 ESC 键响应，同时关闭弹窗内聚焦循环
     */
    closeOnEscape?: boolean;
    /**
     *
     */
    /**
     * 默认：`9999`<br>
     * 通过 zindex 设置弹出框的 z-index，zindex 默认值为 9992,建议不要超过 9999，否则会遮挡住 Dialog 内 Select 等组件的 Popup（默认值 zindex 9999）
     */
    zindex?: any;
    /**
     * 通过 title 设置弹出框标题的提示，title 默认值为 null 如果不传 title，则不显示 title
     */
    titleTip?: string;
    /**
     * 自定义信息弹出框挂载 DOM 的 ID,不传则默认挂载在全局 body 上 例如： mountId='errorDialog'
     */
    mountId?: string;
    hasChecked?: boolean;
    /**
     * 默认：`false`<br>
     * 控制对话最小化属性
     */
    minimizable?: any;
    type?: any;
    detail?: any;
    maxContHeight?: any;
    detailMessage?: any;
    detailMessageShow?: boolean;
    detailStyle?: React.CSSProperties;
    detailMessageTitle?: string;
    content?: any;
    /**
     * 自定义蒙层样式 例如： style={{color:'red'}}
     */
    maskStyle?: React.CSSProperties;
    isMinimized?: boolean;
    /**
     * 默认：`true`<br>
     * 关闭时销毁
     */
    destroyOnClose?: boolean;
    /**
     * 默认：`false`<br>
     * 自定义关闭事件，默认值为 false,此属性用于控制 Dialog 默认关闭按钮，如果为 True，默认点击关闭按钮不生效。
     */
    customClose?: boolean;
    /**
     * 默认：`true`<br>
     * 打开弹层时自动聚焦到关闭按钮
     */
    focusOnClose?: boolean;
    /**
     * 默认：`true`<br>
     * 是否允许弹窗拖出到浏览器窗口之外
     */
    isAllowedExceed?: boolean;
    /**
     *边界数组，控制Dialog的可拖拽范围
     interface Boundary {
      top: number;
      right: number;
      bottom: number;
      left: number;
    }
     */
    boundary?: Boundary;
    onCheckChange?: (isChecked: boolean, event: object) => void;
    /**
     * 默认：`false`<br>
     * 是否允许动画开启
     */
    animationOff?: boolean;
    /**
     * 默认：`true`<br>
     * 是否记录打开弹框时最后一个聚焦点，如果为 `false`, 弹框关闭时候不会聚焦前一个聚焦点
     */
    lastFocus?: boolean;
    /**
     * 是否设置为自定义最小化按钮，设置为 true 后，最小化功能需自己实现。
     */
    customMinimized?: boolean;
    /**
     * 点击最小化按钮的操作事件
     */
    onMinimized?: (event: React.MouseEvent | React.KeyboardEvent) => void;
    iconLocation?: 'content' | 'title';
    /**
     * 默认：`true`<br>
     * 是否自动设置弹窗位置，如果为 `false`，打开弹窗时不会自动调用 setPosition 方法
     */
    autoSetPosition?: boolean;
    /**
     * 默认：`false`<br>
     * 弹窗尺寸变更，是否自动设置弹窗位置居中
     */
    autoCenterOnResize?: boolean;
};
export interface DialogState {
    isOpen?: boolean;
    isMinimize?: boolean;
    moved?: boolean;
    isDisplay?: boolean;
    selected?: boolean;
    dialogOverStyle?: string;
    panelStyle?: any;
    prevLeftVal?: number;
}
export {};
