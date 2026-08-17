import React, { Component } from 'react';
export type labelPosition = 'before' | 'after';
export type spinnerType = 'number' | 'time' | 'customWithPrefixs';
export type timeFormat = 'hh:mm:ss' | 'hh:mm';
export type SpinnerProps = {
    /**
     * 定制组件id，传入该值可覆盖组件自动生成的id 例如： id='eui_spinner_1001'
     */
    id?: string;
    /**
     * 通过class的方式自定义组件样式 例如： className='spinnerClassName'
     */
    className?: string;
    /**
     * 设置是否禁用blur事件对应功能（自动修正错误值）；禁用后onBlur api会失效
     */
    disabledBlurFunction?: boolean;
    /**
     * 自定义微调器样式 例如： style={{color:'red'}}
     */
    style?: React.CSSProperties;
    /**
     * 通过添加class的方式自定义微调器的名称的样式,同时刻控制名称和组件间距
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式自定义微调器的名称的样式,同时刻控制名称和组件间距
     */
    labelStyle?: React.CSSProperties;
    /**
     * 设置微调器的名称文字
     */
    label?: string;
    /**
     * tipStyle 自定义提示框样式
     */
    tipStyle?: React.CSSProperties;
    /**
     * 默认：`before`<br>
     * 设置微调器的名称文字的位置，可选`before`,`after`
     * before代表名称文字在微调器的左边，after代表名称文字在微调器的右边
     */
    labelPosition?: labelPosition;
    /**
     * 当鼠标获取减号的焦点时，显示文本内容 例如：decTitle='减'
     */
    decTitle?: string;
    /**
     * 当鼠标获取加号的焦点时，显示文本内容 例如：incTitle='加'
     */
    incTitle?: string;
    /**
     * 默认：`0`<br>
     * 自定义微调器的最小值，默认为0 例如：min=6
     */
    min?: number | string;
    /**
     * 默认：`100`<br>
     * 自定义微调器的最大值，默认为100 例如：max=10
     */
    max?: number | string;
    /**
     * 默认：`1`<br>
     * 自定义微调器的步伐，默认为1 例如：step=2
     */
    step?: number;
    /**
     * 默认：`0`<br>
     * 自定义微调器的初始值，默认为0 例如：value=8
     */
    value?: string | number;
    /**
     * 默认：`false`<br>
     * 自定义组件是否灰化，默认为false 例如：disabled={true}
     */
    disabled?: boolean;
    /**
     * 自定义微调器文本框值改变时的回调方法(有效值时调用)<br>
     * 签名：`function(value: number) => void`<br>
     * value: 文本框的值
     */
    onChange?: (value: number | string) => void;
    /**
     * 自定义微调器文本框值改变时的回调方法(无效值时调用)<br>
     * 签名：`function(event: object) => void`<br>
     * event: 原生 dom 的 event
     */
    onFocus?: (event: React.FocusEvent) => void;
    /**
     * 自定义微调器文本框失焦时的回调方法
     * 签名：`function(value: number) => void`<br>
     * value: 文本框的值
     */
    onBlur?: (value: number | string) => void;
    /**
     * 默认：`false`<br>
     *  支持设置组件值是否必须，若为必须，则input前的label文字前加*，同时支持非空校验
     */
    required?: boolean;
    /**
     * 默认：`0`<br>
     *  支持设置组件值精度
     */
    precision?: number;
    /**
     * 默认：`number`<br>
     * Spinner 类型,'number','time', 'customWithPrefixs'，分别表示数字型、时间型、自定义前缀类型。
     */
    type?: spinnerType;
    /**
     * 默认：`hh:mm:ss`<br>
     * Specifies the time format for time type spinners. Valid time formats are 'hh:mm:ss','hh:mm'.
     */
    timeFormat?: timeFormat;
    /**
     * 默认：`false`<br>
     * amPm support
     */
    amPm?: boolean;
    /**
     * 通过数组定义取值范围,示例：[[1, 3], [6, 7]]。这个示例表明 1,2,3,6,7 是正确数值. 4 和 5 是错误数值
     */
    rangeArray?: number[][];
    /**
     * 支持组件获取焦点时弹出提示，失去焦点时消失
     */
    focusTip?: string;
    /**
     * 默认：`false`<br>
     * 是否循环显示，如增加到最大值时，继续增加返回至最小值
     * 在设置最大和最小值的情况下使用
     */
    minMaxCycle?: boolean;
    /**
     * 值更新时input组件是否要获取焦点的开关，默认会获取
     */
    doNotFocusWhenValueUpdate?: boolean;
    /**
     * 用户自定义前缀
     */
    customPrefix?: string;
    /**
     * 自定义微调器文本框值改变时的回调方法(无效值时调用)<br>
     * 签名：`function(value: number) => void`<br>
     * value: 文本框的值
     */
    onInputError?: (value: number | string) => void;
    /**
     * 当 type 为'customWithPrefixs'时，点击 Spinner 的加减号的回调函数。<br>
     * 签名：`function(changeTag: string, direction: number,currentValue: string) => void`<br>
     * changeTag: 加减号修改的内容。修改前缀时值为'prefix'，修改内容时值为'content'<br>
     * direction:执行加操作时为 1，减操作时为-1<br>
     * currentValue:输入框的值
     */
    onCustomIncOrDecClick?: (content: string, step: number, value: string | number) => void;
    /**
     * 通过添加 class 的方式自定义微调器输入框的样式
     */
    inputClassName?: string;
    /**
     * 选择开始位置
     */
    selectionStart?: number;
    /**
     * 选择结束位置
     */
    selectionEnd?: number;
    /**
     * 默认：`zh`<br>
     * 当 amPm 为 true 时，设置后缀名是中文（上午/下午）还是英文（AM/PM），如果不设置，
       则使用 ConfigProvider 中配置的语言
     */
    locale?: 'en' | 'zh';
    /**
     * hint 的类型定义
     */
    hintType?: 'div' | 'tip';
    /**
     * 当精度为0时是否将提示信息改为整数
     */
    noZeroPrecise?: boolean;
    onPressEnter?: (value: string | number) => void;
};
type SpinnerState = {
    value: any;
    active: boolean;
    hasValue: boolean;
    hasError: boolean;
    hasFocus: boolean;
    tipWidth: number;
    focusTipDisplay: boolean;
    isScroll: boolean;
    isOnComposition: boolean;
    innerValue: any;
};
export default class Spinner extends Component<SpinnerProps, SpinnerState> {
    static defaultProps: {
        min: number;
        max: number;
        step: number;
        disabled: boolean;
        labelPosition: string;
        required: boolean;
        precision: number;
        type: string;
        timeFormat: string;
        amPm: boolean;
        focusTip: string;
        minMaxCycle: boolean;
        hintType: string;
    };
    static contextType: React.Context<import("react-intl").IntlShape>;
    formatMessage: any;
    cursorPosition: number;
    isincrementAndDecremnt: boolean;
    isTimeBlurFired: boolean;
    blurChangeCursorPosition: number;
    setcursor: {
        start: number;
        end: number;
    };
    userIsTyping: boolean;
    numMax: string | number;
    numMin: string | number;
    cursorState: {
        start: number;
        end: number;
    };
    inputRef: HTMLInputElement;
    tip: any;
    id: string;
    constructor(props: SpinnerProps, context: any);
    handleScroll: (event: any) => boolean;
    getValueByProps(): string | number;
    componentWillReceiveProps(nextProps: SpinnerProps): void;
    componentDidUpdate(prevProps: SpinnerProps, prevState: SpinnerState): void;
    getValue(): any;
    checkTimeFormat(value: any): any;
    /**
     * To allow valied keys for the time spinner .
     */
    checkValidKey: (key: number) => boolean;
    /**
     * To allow valid keys for number spinner
     */
    validKey: (key: number) => boolean;
    handlePrecisionedValue: (value: string, precision: number) => string;
    isNumber: (obj: any) => boolean;
    handleChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    handleFocus: (event: React.FocusEvent<HTMLInputElement>) => void;
    handleBlur: (e: any, type?: string) => void;
    handleParentMethod: (val: number | string, type: string | undefined) => void;
    getIncValueByRangArr: (calcResult: any, rangeArray: number[][]) => any;
    getDecValueByRangArr: (calcResult: any, rangeArray: number[][]) => any;
    /**
     * 增加按钮点击事件
     * @returns
     */
    handleIncClick: () => void;
    /**
     * 减少按钮点击事件
     * @returns
     */
    handleDecClick: () => void;
    handleKeys: (e: React.KeyboardEvent<HTMLInputElement>) => void;
    handleCustomPrefixKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
    onChangeEvent: (result: string | number) => void;
    handleMouseDown: (event: React.MouseEvent) => void;
    handleMouseUp: (event: React.MouseEvent) => void;
    isValid: (value: string | number) => boolean;
    getValidateResult: (val: number | string) => boolean;
    handleTimeMouseUp: (e: React.MouseEvent<HTMLInputElement, MouseEvent>) => void;
    handleCustomPrefixMouseUp: (e: React.MouseEvent<HTMLInputElement, MouseEvent>) => void;
    handleContainerBlur: () => void;
    /**
     * On Key down we are restrict the alpha and specia chars .
     */
    handlePaste: (event: any) => void;
    handleStopPrevents: (event: any) => void;
    handleKeyDown: (event: any) => void;
    validate(): boolean;
    handleCustomInputChange: (e: any) => void;
    handleComposition: (e: any) => void;
    onInputNumber: (e: any) => void;
    onInputTime: (e: any) => void;
    render(): React.JSX.Element;
}
export {};
