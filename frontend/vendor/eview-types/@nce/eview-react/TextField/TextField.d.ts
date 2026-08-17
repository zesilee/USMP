import React, { Component } from 'react';
export type AutoCompleteType = 'off' | 'on' | 'new-password';
export type TextFieldProps = {
    /**
     *  组件默认设置动态id，且组件支持用户自定义id
     */
    id?: string;
    /**
     * 通过添加style的方式自定义该组件的整体样式（作用于最外层div）
     * 例如：margin padding
     */
    style?: React.CSSProperties;
    /**
     * 通过添加class的方式自定义该组件的整体样式（作用于最外层div）
     * 例如：margin padding
     */
    className?: string;
    /**
     * 设置输入框的名称文字
     */
    label?: string;
    /**
     * 设置文本输入框的placeholder
     */
    placeholder?: string;
    /**
     * 通过添加class的方式控制label的样式以及输入框和文本之间的间距
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式控制label的样式以及输入框和文本之间的间距
     */
    labelStyle?: React.CSSProperties;
    labelTitle?: string;
    /**
     * Control the style of the error tip label .
     */
    /**
     * 该属性是通过添加class的方式来控制input的样式
     */
    inputClassName?: string;
    /**
     * 该属性是通过添加style的方式来控制input的样式,比如给input设置宽度
     */
    inputStyle?: React.CSSProperties;
    /**
     *  支持给组件设置value值
     */
    value?: string;
    /**
     * 默认：`before`<br>
     * 设置输入框的名称文字的位置
     * before代表名称文字在输入框的左边，after代表名称文字在输入框的右边
     */
    labelPosition?: 'before' | 'after';
    /**
     * 默认：`text`<br>
     * 指定要显示的输入类型 "password" 或者 "text". 默认值为文本类型.
  注意：如果类型为密码，则 BM 需要传递自动完成属性。 请参阅链接中的自动完成准则
  https://developer.mozilla.org/en-US/docs/Web/Security/Securing_your_site/Turning_off_form_autocompletion
     */
    type?: 'password' | 'text';
    /**
     * 默认：`false`<br>
     *  支持设置组件值是否必须，若为必须，则input前的label文字前加*，同时支持非空校验
     */
    required?: boolean;
    /**
     * 默认：`false`<br>
     *  支持组件在设置了required属性后，将红色的*标记隐藏
     */
    hideRequiredMark?: boolean;
    /**
     * 默认：`false`<br>
     * 设置组件的灰化
     */
    disabled?: boolean;
    /**
     * 设置输入框是否展示title
     */
    /**
     * 默认：`true`<br>
     * If true then Select the text when onFocus is called.
     */
    selectOnfocus?: boolean;
    /**
     * If below function return false then entered character will not be considered.<br>
     * 签名：`function(value: value, id: id) => Object`<br>
     * value: The value entered by a user<br>
     * id: ID of component which generate this.<br>
     */
    isCharacterAllowed?: (value: string, id: string) => boolean;
    /**
     * 设置自定义校验规则<br>
     * 签名:`function(value: value,id: id, type: type) => Object`<br>
     * value: 用户输入值<br>
     * id: ID of component which generate this.<br>
     * type: if its triggered by "onChange" or "onBlur"<br>
     * returns (Object): validationReturnMap={'result'：result,'message':message}<br>
     * result {bool} 用来设置校验是否有错误<br>
     * message {string} 用来设置错误提示的文本信息<br>
     * result和message，不能更换为其他字段<br>
     * TextField 组件的静态属性 defaultValidator 提供了下列默认校验函数：min, max, range, number, email, digit, url, alpha, regex, postfix, ipv4, ipv6, creditCard, equalTo, notEqualTo, minLength, maxLength, rangeLength。例如，如果希望用户输入的最小值是 5, 可以这样使用 <TextField validator={TextField.defaultValidator.min(5)}/>
     */
    validator?: (value: string, id?: string, type?: string) => {
        result: boolean;
        message: string | React.JSX.Element;
        type?: 'error' | 'tip';
    };
    /**
     * 空值的时候是否执行校验
     */
    validateWhileEmpty?: boolean;
    /**
     * 定义组件的失去焦点时的事件<br>
     * 签名：`function(event: object) => void`<br>
     * event 原生dom的event
     */
    onBlur?: (event: React.FocusEvent, value: any) => void;
    /**
     * 定义组件内容改变时的事件<br>
     * 签名：`function(value1: string, value2: string, event: object) => void`<br>
     * value1: new value<br>
     * value2: previous value<br>
     * event: 原生 dom 的 event
     */
    onChange?: (value: string, oldValue: string | number, event: React.ChangeEvent) => void;
    /**
     * 组件点击事件的回调函数
     */
    onClick?: (event: any) => void;
    /**
     * 定义组件聚焦时的事件<br>
     * 签名：`function(event: object) => void`<br>
     * event 原生dom的event
     */
    onFocus?: (event: React.FocusEvent) => void;
    /**
     * 定义组件在按下键盘键时的事件<br>
     * 签名：`function(event: object) => void`<br>
     * event 原生dom的event
     */
    onKeyDown?: (event: React.KeyboardEvent<HTMLInputElement>) => void;
    /**
     * 定义组件在放开键盘键时的事件<br>
     * 签名：`function(event: object) => void`<br>
     * event 原生dom的event
     */
    onKeyUp?: (event: React.KeyboardEvent) => void;
    /**
     * 定义组件粘贴的事件
     */
    onPaste?: (event: React.ClipboardEvent<HTMLInputElement>) => void;
    /**
     * 默认：`any`<br>
     * 设置输入类型，当 format 为 number 时，只能输入数字,
     * 当不设置该属性或为 any 时，可以输入任意值
     */
    format?: 'number' | 'any';
    /**
     * 默认：`div`<br>
     * hint 的类型定义
     */
    hintType?: 'div' | 'tip';
    /**
     * 默认：`false`<br>
     * Readonly
     */
    readOnly?: boolean;
    /**
     * 支持组件获取焦点时弹出提示，失去焦点时消失
     */
    focusTip?: string;
    /**
     * 默认：`false`<br>
     * 用户同时配置了FocusTip且Validator报错的时候，想同时显示这两种信息
     */
    showFocusTipAndError?: boolean;
    /**
     * 默认：`9999`<br>
     * 支持组件自定义提示框 z-index
     */
    zIndex?: number;
    /**
     * 自动生成zindex，取页面中当前最高；谨慎使用，对性能有消耗
     */
    autoZindex?: boolean;
    /**
     * 提示气泡持续时间(ms)
     */
    /**
     * 默认：`off`<br>
     * autoComplete 是否填充表单,可选 `off` , `on` , `new-password`
     */
    autoComplete?: AutoCompleteType;
    /**
     * 通过添加 tipStyle 来自定义悬浮提示框（TipBox）的样式acterAllowed
     */
    tipStyle?: React.CSSProperties;
    /**
     * 设置文本域中最大的可输入长度,超过长度将不能继续输入
     */
    maxLength?: number;
    maxLengthByte?: boolean;
    /**
     * ev_textField_container的区域样式
     */
    containerStyle?: React.CSSProperties;
    /**
     * 默认：`false`<br>
     * 是否允许复制粘贴 密码
     */
    canPasswordPaste?: boolean;
    /**
     * title
     */
    title?: string;
    /**
     * 默认：`false`<br>
     * 是否支持password场景下，根据props修改value的值
     * 原来由于安全原因不支持此功能，开启此功能后由业务自行处理
     */
    isAllowToModifyPasswordByProps?: boolean;
    /**
     * 设置输入框右边显示的输入标准（该提示一直存在于输入框右侧）
     */
    ruleText?: string;
    /**
     * 默认：`none`<br>
     * label与输入框的间隔大小
     */
    enableFixWidth?: 'small' | 'middle' | 'large' | 'none';
    /**
     * 默认：`false`<br>
     */
    enableFocusAntiShake?: boolean;
    /**
     * 默认：`false`<br>
     * 错误信息的优先级最高
     */
    errorLevelMax?: boolean;
    /**
     * 后缀元素
     */
    suffix?: React.ReactNode;
};
interface TextFeildState {
    hasFocus: boolean;
    isScroll: boolean;
    hasValue: boolean;
    value: string;
    hasError: boolean;
    capsON: boolean;
    innerValue: string;
    /**
     * 在输入框内输入拼音的时候，在拼音上位输入完成时，字母会触发Input的onChange事件。
     * 如果想在只有拼完后才触发相应的时间，就要用到onCompositionStart以及onCompositionEnd事件了。
     */
    isOnComposition: boolean;
    /**
     * 判断是否是chrome浏览器或者火狐浏览器
     */
    isChrome: boolean;
}
export default class TextField extends Component<TextFieldProps, TextFeildState> {
    static defaultProps: {
        labelPosition: string;
        required: boolean;
        hideRequiredMark: boolean;
        disabled: boolean;
        type: string;
        selectOnfocus: boolean;
        format: string;
        hintType: string;
        readOnly: boolean;
        focusTip: string;
        showFocusTipAndError: boolean;
        zIndex: number;
        isAllowToModifyPasswordByProps: boolean;
        autoComplete: string;
        canPasswordPaste: boolean;
        enableFocusAntiShake: boolean;
        errorLevelMax: boolean;
    };
    static defaultValidator: {
        min: (num: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        max: (num: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        integer: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        range: (min: number, max: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        rangeAndInteger: (min: number, max: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        number: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        email: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        digit: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        url: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        alpha: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        regex: (reg: string) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        postfix: (str: string) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        ipv4: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        ipv6: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        creditCard: () => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        equalTo: (str: string) => (value: string) => {
            result: boolean; /**
             * 定义组件聚焦时的事件<br>
             * 签名：`function(event: object) => void`<br>
             * event 原生dom的event
             */
            message: React.JSX.Element;
        };
        notEqualTo: (str: string) => (value: string) => {
            result: boolean; /**
             * 定义组件在按下键盘键时的事件<br>
             * 签名：`function(event: object) => void`<br>
             * event 原生dom的event
             */
            message: React.JSX.Element;
        };
        minLength: (length: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        maxLength: (length: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
        rangeLength: (minLength: number, maxLength: number) => (value: string) => {
            result: boolean;
            message: React.JSX.Element;
        };
    };
    inputDom: any;
    tip: any;
    id: string;
    validityEntity: any;
    state: {
        hasFocus: boolean;
        hasValue: boolean;
        isScroll: boolean;
        value: string;
        hasError: boolean;
        capsON: boolean;
        innerValue: string;
        isOnComposition: boolean;
        isChrome: boolean;
    };
    compositionStatus: string;
    focusAntiShakeFlag: boolean;
    static contextType: React.Context<import("react-intl").IntlShape>;
    formatMessage: any;
    validateType: string;
    errorMsg: string;
    needValidate: boolean;
    removePasswordTimeOut: any;
    suffix: HTMLDivElement;
    constructor(props: TextFieldProps, context: any);
    handleComposition: (e: any) => void;
    /**
     *
     * @param event
     * 1、123，选中23，输入a，value值为1a
     * 注意，firefox浏览器下输入中文时，通过数字选择最终触发的中文时，不会触发输入框的onChange事件，但是chrome会触发
     * firefox下偶尔会触发，暂未发现规律
     */
    handleInputChange: (event: any) => void;
    handleInputBlur: (event: any) => void;
    handleInputFocus: (event: any) => void;
    handleLabelClick: () => void;
    handleCapsLock: (e: any) => void;
    isValid: (value: any) => boolean;
    componentWillReceiveProps(nextProps: any): void;
    getValue(): any;
    /**
     * Tipbox触发滚动的时候，会回调该函数
     * @param event
     * @returns
     */
    handleScroll: (event: any) => boolean;
    componentDidMount(): void;
    removePasswordValueAttribute: () => void;
    getOldValue(): string;
    getCapsLockTipText(): React.JSX.Element;
    validate: () => boolean;
    focus(): void;
    handleKeyDown: (event: any) => void;
    /**
     * 处理组件内Input的KeyDown事件
     * 这里好像没哟必要处理isCharacterAllowed，虽然这里
     * 而且ctrl+v的时候没法处理，因为ctrl+v的时候，因为此时event.key是v，而不是粘贴面板的值
     * @param event
     */
    handleInputKeyDown: (event: any) => void;
    componentDidUpdate: () => void;
    setSuffixPosition: () => void;
    handleInputClick: (event: any) => void;
    handleMaxLength: (event: any) => void;
    pasteMethod: (event: any) => void;
    render(): React.JSX.Element;
}
export {};
