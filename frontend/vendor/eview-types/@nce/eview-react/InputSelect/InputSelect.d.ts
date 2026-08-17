import React, { Component } from 'react';
import { PopupProps } from '../Popup';
export type InputSelectProps = {
    /**
     * 设置传入数据let options = [{ text: "111", value: 1 }, {text: "222", value: 2}, {text: "333",value: 2}];
     * <br>其中text是页面展示值，value是数据库存取值
     */
    options: any;
    /**
     * 设置Select的id
     */
    id?: string;
    /**
     * 通过自定义style来改变组件的的整体样式
     * 例如:margin padding display
     */
    style?: React.CSSProperties;
    /**
     * 通过自定义class来改变组件的整体样式
     * 例如:margin padding display
     */
    className?: string;
    /**
     * 通过添加class的方式自定义label的名称的样式
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式自定义label的名称的样式
     */
    labelStyle?: React.CSSProperties;
    /**
     * 通过自定义class来改变select的样式
     * 例如:border color background width height
     */
    selectClassName?: string;
    /**
     * 通过自定义style来改变select的样式
     * 例如:border color background width height
     */
    selectStyle?: React.CSSProperties;
    /**
     * 通过自定class来改变option的样式
     */
    optionClassName?: string;
    /**
     * 通过自定style来改变option的样式
     */
    optionStyle?: any;
    /**
     * 设置下拉框的名称文字
     */
    label?: string;
    /**
     * 默认：`before`<br>
     * 设置组件的名称文字的位置
     * before代表名称文字在组件的左边，after代表名称文字在组件的右边
     */
    labelPosition?: 'before' | 'after';
    /**
     * 设置选中的index,此处为数据属性options中对应的index，优先级高于value
     */
    selectedIndex?: number;
    /**
     * 通过设置value值来选中select中的对应项
     */
    value?: any;
    /**
     * 默认：`false`<br>
     * 设置组件的灰化
     */
    disabled?: boolean;
    /**
     * 默认：`false`<br>
     * 设置组件是否为必填项
     */
    required?: boolean;
    /**
     * 默认：`true`<br>
     *Enable or disable the search tip "Not Found"
     */
    showSearchTip?: boolean;
    /**
     * 默认：`false`<br>
     *Enable or disable the horizontal Scrollbar
     */
    enablHorzScroll?: boolean;
    /**
     * 设置inputSelect的onChange事件,监听inputSelect选中值得变化<br>
     * 签名：`function(value: any, oldValue: any) => void`<br>
     * value: 当前选中的选项的value值
     * oldValue: 组件上一次算中值
     * type: 'input' | 'select' 区分onChange事件的触发主题
     */
    onChange?: any;
    onSelect?: (value: any, oldValue: any) => void;
    /**
     * 设置 inputSelect 的 onFocus 事件<br>
     * 签名：`function(e: object) => void`<br>
     * event: 原生dom的聚焦事件
     */
    onFocus?: (event: React.MouseEvent) => void;
    /**
     * 设置 inputSelect 的 onBlur 事件<br>
     * 签名：`function(e: object) => void`<br>
     * event: 原生dom的聚焦事件
     */
    onBlur?: (event: React.MouseEvent) => void;
    /**
     * return the  value of input on Enter press.<br>
     * 签名：`function(e: object, value) => void`<br>
     * _e:_ Primary dom events<br>
     * value: value entered
     */
    onInputEnter?: any;
    /**
     * inputSelect on inputbox keyUp event should fire this call back.<br>
     * 签名：`function(value) => void`<br>
     * value: value entered
     */
    onInputKeyUp?: any;
    /**
     * 默认：`bottom`<br>
     * Defines the direction of the popup in selection control
     */
    popupDirection?: 'top' | 'bottom';
    /**
     * 默认：`9999`<br>
     * To set the zindex popup .
     */
    zindex?: string;
    /**
     *  Popup close callback
     */
    onClosePopup?: any;
    /**
     * popup关闭回调
     */
    onOpenPopup?: any;
    /**
     * 默认：`div`<br>
     * hint 的类型定义
     */
    hintType?: 'div' | 'tip';
    /**
     * 支持组件获取焦点时弹出提示，失去焦点时消失
     */
    /**
     * 设置自定义校验规则<br>
     * 签名：`function(value: value,id: id, type: type) => Object`<br>
     * value: 用户输入值<br>
     * id: ID of component which generate this.<br>
     * type: if its triggered by "onChange" or "onBlur"<br>
     * returns (Object): validationReturnMap={'result'：result,'message':message}<br>
     * result {bool} 用来设置校验是否有错误<br>
     * message {string} 用来设置错误提示的文本信息<br>
     * result和message，不能更换为其他字段<br>
     * TextField 组件的静态属性 defaultValidator 提供了下列默认校验函数：min, max, range, number, email, digit, url, alpha, regex, postfix, ipv4, ipv6, creditCard, equalTo, notEqualTo, minLength, maxLength, rangeLength。例如，如果希望用户输入的最小值是 5, 可以这样使用 <TextField validator={TextField.defaultValidator.min(5)}/>
     */
    validator?: (value: string, id?: string, onChange?: string) => {
        result: boolean;
        message: string | Element;
    };
    /**
     * 默认：`true`<br>
     * Filters the list according to the specified case-insensitive value typed in input select component.
     */
    caseInsensitiveFilter?: boolean;
    /**
     *{autoComplete:'off'}
     */
    inputProps?: React.InputHTMLAttributes<HTMLElement>;
    /**
     * 默认：`false`<br>
     *  输入框输入的值只会用于过滤搜索，Input失焦的时候会清空，只要从下拉选项中选中值才能被保留
     */
    onlySelect?: boolean;
    /**
    * 设置是否自定义宽度，为true时，宽度固定，支持单行省略，中英文切换时自动切换宽度？
    */
    enableFixWidth?: 'small' | 'middle' | 'large' | 'none';
    virtualScroll?: boolean;
    /**
    * 鼠标离开tipbox后是否关闭tipbox
    */
    isMouseLeaveClose?: boolean;
    /**
  * 设置文本输入框的placeholder
  */
    placeholder?: string;
    /**
     * 为true时，模糊搜索会去掉头尾空格
     */
    searchTrim?: boolean;
    /**
     * 透传下拉框popUp的属性
     */
    popUpProps?: PopupProps;
    /**
     * 再次打开下拉的时候保持搜索过滤结果
     */
    keepFiter?: boolean;
    /**
     * 默认：`false`<br>
     * 支持清除按钮
     */
    enableClear?: boolean;
    onClear?: any;
    onlySelectLastValue?: boolean;
    /**
     * 是否执行组件更新，在componentWillReceiveProps生命周期中将控制权移交给用户
     * @returns boolean
     */
    shouldRender?: (stateValue: InputSelectProps['value'], propsValue: InputSelectProps['value']) => boolean;
};
interface InputSelectState {
    hasFocus: boolean;
    isScroll: boolean;
    hasValue: boolean;
    /**最终input显示的值 */
    value: any;
    hasError: boolean;
    focus: boolean;
    selectedIndex: number;
    data: any;
    /**存储option选项和输入框的模糊匹配程度，true表示匹配（即要在下拉列表中显示） */
    dataStatus: any;
}
export default class Select extends Component<InputSelectProps, InputSelectState> {
    static defaultProps: {
        disabled: boolean;
        required: boolean;
        labelPosition: string;
        showSearchTip: boolean;
        zindex: string;
        hintType: string;
        caseInsensitiveFilter: boolean;
        onlySelect: boolean;
        options: any[];
        virtualScroll: boolean;
        enableClear: boolean;
    };
    index: number;
    allDisable: boolean;
    sltIpt: any;
    id: any;
    pop: any;
    allDisabled: boolean;
    isBlurFiredAlready: boolean;
    timeoutUpdate: number;
    tip: any;
    dom: HTMLDivElement;
    optionClickFlag: boolean;
    select: HTMLDivElement;
    selectWidth: any;
    formatMessage: any;
    static contextType: React.Context<import("react-intl").IntlShape>;
    selectedValue: any;
    isInputChanged: boolean;
    inputValueForOnlySelect: any;
    constructor(props: any, context: any);
    getValue(): any;
    setValue(value: any): void;
    getSelectedValue(): any;
    validate(): boolean;
    focus(): void;
    clear: () => void;
    isChild: (element: any) => boolean;
    handleOptionClick: (data: any) => void;
    isAllDisable: () => void;
    handleKeyPress: (e: any) => void;
    onEscHandle: (date: any) => void;
    handleSelectClick: (e: any) => void;
    handleKeyPressEvent: (event: any) => void;
    handleKeyUpEvent: (event: any) => void;
    handleInputChange: (e: any) => void;
    handleInputFocus: (e: any) => void;
    handleInputblur: (e: any) => void;
    handleBlur: (e: any) => void;
    /**
     * tipbox和popup触发滚动的时候，都会回调该函数
     * @param e
     */
    handleScroll: (e: any) => void;
    getIndexByValue(value: any, data: any): any;
    getSelectedIndex(selectedIndex: any, value: any, data: any): any;
    escapeRegExp(str: any): any;
    collapseDropDown(): void;
    /**
     * 模糊搜索,获取下拉列表中和Value匹配的值，匹配就是true,不匹配就是false
     * @param param 需要模糊匹配的字符串
     */
    getDataStatus(param: any, newOptions?: any): any[];
    validateParam: (param: any) => boolean;
    componentWillReceiveProps(nextProps: any): void;
    componentDidUpdate(): void;
    isValid: (value: any) => boolean;
    render(): React.JSX.Element;
}
export {};
