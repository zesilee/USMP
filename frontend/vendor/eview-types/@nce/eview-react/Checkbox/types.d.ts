export type CheckboxProps = {
    /**
     * 设置Checkbox的id
     */
    id?: string;
    /**
     * 设置组件的name
     */
    name?: string;
    /**
     * 通过自定义类的方式控制checkbox的整体样式（最外层div）
     * 例如?:margin、padding
     */
    className?: string;
    /**
     * 通过自定义style的方式控制checkbox的整体样式（最外层div）
     * 例如?:margin、padding
     */
    style?: object;
    /**
     * 通过自定义类的方式控制checkbox的图标尺寸以及颜色
     */
    checkboxClassName?: string;
    /**
     * 通过自定义style的方式控制checkbox的图标尺寸以及颜色
     */
    checkboxStyle?: object;
    /**
     * 通过添加class的方式控制label的样式,以及checkbox与文本之间的间距
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式控制label的样式,以及checkbox与文本之间的间距
     */
    labelStyle?: object;
    /**
     * 设置Checkbox的展示文本值
     */
    label?: string;
    /**
     * 设置Checkbox的在数据库中存取的value
     */
    value?: any;
    /**
     * 默认：`after`<br>
     * 设置Checkbox的文本值在Checkbox图标的左侧还是右侧
     * before代表文本在Checkbox的左边，after表示文本在Checkbox的右边
     */
    labelPosition?: 'before' | 'after';
    /**
     * 默认：`false`<br>
     * 设置checkbox的灰化
     */
    disabled?: boolean;
    /**
     * 默认：`false`<br>
     * 设置Checkbox的是否是选中状态
     */
    checked?: boolean;
    /**
     * 默认：`false`<br>
     * 设置Checkbox的拥有半选状态
     */
    halfChecked?: boolean;
    /**
     * 检查是否需要继续操作。此函数将在更改 onChange 之前调用。<br>
     * 签名：`(value: any, check: boolean, event: any) => boolean;`<br>
     * value:用户在properties中传入的value值<br>
     * check (checked or unchecked)<br>
     * event:原生dom的点击事件
     */
    onPreChange?: (value: any, check: boolean, event: any) => boolean;
    /**
     * 设置Checkbox的onChange事件<br>
     * 签名：`(value: any, check: boolean, event: any, additionalData: object) => void;`<br>
     * value 用户在properties中传入的value值<br>
     * check 代表了复选框是否选中<br>
     * event 原生dom的点击事件
     */
    onChange?: (value: any, check: boolean, event: any, additionalData: object) => void;
    /**
     * 设置Checkbox的聚焦事件<br>
     * 签名：`function(value: any, checked: bool, event: object) => void`<br>
     * value 用户在properties中传入的value值<br>
     * check 代表了复选框是否选中<br>
     * event 原生dom的聚焦事件
     */
    onFocus?: (value: any, check: boolean, event: any) => void;
    /**
     *You can pass additional data from upper level component and get it on clicking the checkbox
     */
    additionalData?: object;
    /**
     *tipText is the text to dispaly in the tip
     */
    tipText?: string;
    /**
     *Additional props provided for the tip like disposeTimeOut, arrowDirection etc. tipData={{disposeTimeOut:5000,arrowDirection:'left',style:{maxWidth:'218px'}}}}
     */
    tipData?: object;
    /**
     *定制方框的tabindex
     */
    boxTabIndex?: string;
    /**
     * 设置 Checkbox 的 onBlur 事件<br>
     * 签名：`(value: any, check: boolean, event: any) => void;`<br>
     * value 用户在properties中传入的value值<br>
     * check 代表了复选框是否选中<br>
     * event 原生dom的失焦事件
     */
    onBlur?: (value: any, check: boolean, event: any) => void;
    /**
     * tip的样式
     */
    tipStyle?: React.CSSProperties;
    /**
     * 用在tree组件，避免props.checked为true被state.checked覆盖的情况
     */
    treeChecked?: boolean;
    /**
     *用在tree组件，避免props.halfChecked为true被state.halfChecked覆盖的情况
     */
    treeHalfChecked?: boolean;
    forceUpdate?: boolean;
    onMouseMove?: any;
    /**
     *自定义label后缀
     */
    customSuffixContent?: React.ReactNode | string;
};
export interface CheckboxState {
    focus: boolean;
    halfChecked: boolean;
    checked: boolean;
    displayTip: boolean;
}
