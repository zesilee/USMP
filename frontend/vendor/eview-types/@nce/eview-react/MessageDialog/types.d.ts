export type MessageDialogProps = {
    /**
     *定制组件dom元素id，传入该值可覆盖组件自动生成的id
     * 例如：
     * id='MessageDialogId'
     */
    id?: string;
    /**
     * 定制组件dom元素className，传入该值可追加组件自动生成的className，从而自定义组件样式
     * 例如：
     * className='MessageDialogClassName'
     */
    className?: string;
    /**
     * 定制信息提示框标题，不传则标题为对应类型的默认值
     * 例如：
     * title='MessageDialog'
     */
    title?: string;
    /**
     * 自定义信息提示框业务信息提示标题，不传则为空
     * 例如：
     * content='Request Failed ！'
     */
    content?: string;
    /**
     * 自定义信息提示框业务信息提示详细，不传则为空
     * 例如：
     * <br/>detail='Request data exception'
     * <br/>detail=&lt;div&gt;这是详细信息&lt;/div&gt;
     */
    detail?: any;
    detailStyle?: object;
    /**
     * 自定义提示框按钮文本,以及回调
     * 例如：buttons={
     * <br/>ok?:{text?:'确认',onClick?:()=>{确认回调},
     * <br/>cancel?:{text?:'取消',onClick?:()=>{alert(取消回调)}}}}
     * <br>`optional` focused :设置焦点在焦点索引按钮上。只针对一个按钮设置. 下面是一个例子：cancel:{text:'Cancel',focused: true, onClick:()=>{alert(Canceling callback)}}
     */
    buttons?: object;
    /**
     * 默认：`info`<br>
     * 必填项，定制信息提示框的提示类型，仅有四种枚举类型可传入
     * <br/>例如：
     * type='error'
     */
    type?: 'error' | 'info' | 'success' | 'warn' | 'confirm' | 'risk' | 'highRisk';
    /**
     * 自定义信息提示框样式
     * 例如：
     * style={{color?:'red'}}
     */
    style?: object;
    /**
     * 默认：`false`<br>
     * 自定义信息提示框是否显示，默认值为false（关闭）
     * 例如：
     * isOpen={true}
     */
    isOpen?: boolean;
    /**
     * 默认：`true`<br>
     * 自定义信息提示框是否显示关闭按钮，默认值为true（可关闭）
     * 例如：
     * closable={false}
     */
    closable?: boolean;
    /**
     * 默认：`true`<br>
     * 自定义信息提示框是否模态，不传，则默认为true
     * 例如：
     * modal={false}
     */
    modal?: boolean;
    /**
     * 自定义信息提示框大小，两个数字的数组[x,y]，x标识宽度，y标识高度，不传则使用默认大小
     * <br/>例如：size=[300,300]
     * <br/>只设置宽度，不设置高度时，例如：[300，'auto']
     * <br/>只设置高度，不设置宽度时，例如：['auto'，300]
     * <br/>The minimum width is 350px.
     * <br/>The minimum height is 240px.
     */
    size?: any;
    /**
     * 自定义信息提示框显示位置，两个数字的数组[x,y]，x标识宽度，y标识高度，不传则使用默认位置
     * <br/>例如：position：[200,200]
     * <br/>只设置左边距，不设置右边距时，例如：[300，'auto']
     * <br/>只设置右边距，不设置左边距时，例如：['auto'，300]
     */
    position?: any;
    /**
     * Max Height of the messageDialog content.
     * If height is greater that maxContHeight then height equals to maxContHeight.
     */
    maxContHeight?: number;
    /**
     * 自定义点击关闭按钮关闭信息提示框时的回调方法，不传则无回调<br>
     * 签名：`function(event: object) => void`<br>
     * event: 原生dom事件
     */
    onClose?: (event: object) => void;
    /**
     * 显示详细信息
     */
    detailMessage?: object;
    /**
     * 默认：`9993`<br>
     * Tio set the zindex value by user .
     */
    zindex?: string;
    /**
     * 默认：`body`<br>
     * 自定义信息弹出框挂载 DOM 的 ID,不传则默认挂载在全局 body 上 例如： mountId='errorDialog'
     */
    mountId?: string;
    /**
     * 当 type 为 risk highRisk 时控制 CheckBox 勾选
     */
    hasChecked?: boolean;
    /**
     * CheckBox切换事件
     */
    onCheckChange?: (isChecked: boolean, event: object) => void;
    /**
     * 默认：`false`<br>
     * 是否允许动画开启
     */
    animationOff?: boolean;
    /**
     * 默认：`详情`<br>
     * 显示 detailMessage 时候的标题
     */
    detailMessageTitle?: string;
    /**
     * 默认：`false`<br>
     * detailMessage默认是否展开
     */
    detailMessageShow?: boolean;
    /**
     * 默认：`content`<br>
     * 控制弹框图标的位置
     */
    iconLocation?: 'content' | 'title';
    /**
     * 默认：`false`<br>
     * 是否自动设置弹框位置
     */
    autoSetPosition?: boolean;
};
export type MessageDialogState = {};
