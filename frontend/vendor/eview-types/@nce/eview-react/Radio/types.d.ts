export type RadioProps = {
    /**
     * 设置radio的id
     */
    id?: string;
    /**
     * 通过自定义class改变radio的整体样式,作用于最外层div
     * 例如：margin、padding、font
     */
    className?: string;
    /**
     * 通过自定义style改变radio的整体样式,作用于最外层div
     * 例如：margin、padding、font
     */
    style?: object;
    /**
     * 通过添加class的方式控制label的样式以及文本值与radio之间的间距
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式控制label的样式以及文本值与radio之间的间距
     */
    labelStyle?: object;
    /**
     * 通过自定义class改变radio的样式
     * 例如：颜色、尺寸
     */
    radioClassName?: string;
    /**
     * 通过自定义style改变radio的样式
     * 例如：颜色、尺寸
     */
    radioStyle?: object;
    /**
     * 设置radio的展示文本值
     */
    label?: string;
    /**
     * 默认：`after`<br>
     * 设置radio的文本值在radio图标的左侧还是右侧
     * before代表文本在radio的左边，after表示文本在radio的右边
     */
    labelPosition?: 'before' | 'after';
    /**
     * 设置radio的存入数据库的value值
     */
    value?: any;
    /**
     * 默认：`false`<br>
     * 设置radio是否选中
     */
    checked?: boolean;
    /**
     * 默认：`false`<br>
     * 设置radio是否灰化
     */
    disabled?: boolean;
    /**
     * 设置radio的选中事件<br>
     * 签名：`function(value: any, event: object) => void`<br>
     * {any} value 用户在properties中传入的value值<br>
     * {object} event 原生dom的点击事件
     */
    onChange?: (value: any, event: object) => void;
    /**
     * 设置radio的聚焦事件<br>
     * 签名：`function(value: any, event: object) => void`<br>
     * {any} value 用户在properties中传入的value值
     * {object} event 原生dom的聚焦事件
     */
    onFocus?: (value: any, event: object) => void;
    /**
     * TipText 是 Tip 中显示的文本
     */
    tipText?: string;
    /**
     *额外的道具提供给小费,如 disposeTimeOut ,箭头方向等. tipData = {{disposeTimeOut : 5000 ,箭头方向: ' left '}}
     */
    tipData?: object;
    /**
     *props given for the label description
     */
    description?: string;
    onBlur?: (value: any, event: object) => void;
    title?: string;
    /**
     * 设置为 受控组件
     */
    isControlled?: boolean;
};
export interface RadioState {
    focus: boolean;
    checked: boolean;
    displayTip: boolean;
}
