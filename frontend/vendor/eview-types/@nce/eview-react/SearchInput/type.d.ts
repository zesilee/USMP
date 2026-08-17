type labelPositionType = 'before' | 'after';
type popItemsType = {
    text: string;
    value: any;
    disabled?: boolean;
};
export type SearchInputProps = {
    /**
     *  搜索输入框根节点的id。
     */
    id?: string;
    /**
     *  搜索输入框根节点的样式类名。
     */
    className?: string;
    /**
     * 覆盖搜索输入框根节点的行内样式。
     */
    style?: React.CSSProperties;
    /**
     * 默认：`false`<br>
     * 指定其必填字段
     */
    required?: boolean;
    /**
     *  支持组件在设置了required属性后，将红色的*标记隐藏
     * @defaultValue false
     */
    hideRequiredMark?: boolean;
    /**
     * 默认：`false`<br>
     * 设置组件的灰化
     */
    disabled?: boolean;
    /**
     * 输入框的样式类名。
     */
    inputClassName?: string;
    /**
     * 覆盖搜索框的行内样式。
     */
    inputStyle?: React.CSSProperties;
    /**
     * 通过添加class的方式自定义label的名称的样式
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式自定义label的名称的样式
     */
    labelStyle?: React.CSSProperties;
    /**
     * 设置下拉框的名称文字
     */
    label?: string;
    /**
     * 默认：`before`<br>
     * 设置输入框的名称文字的位置,可选`before`,`after`
     * before代表名称文字在输入框的左边，after代表名称文字在输入框的右边
     */
    labelPosition?: labelPositionType;
    /**
     * 搜索输入框的文本值。
     */
    value?: any;
    /**
     * 下拉列表的内容，和onSuggest互斥，两者不能同时设置，格式如下：
     * <br>[{
     * <br>text: "a",      * <br>value: 1      * <br>}, ...]
     */
    popItems?: popItemsType[];
    /**
     * 搜索输入框的占位语。
     */
    placeholder?: string;
    /**
     * To limit the charcter in search input box
     */
    maxLengthInput?: number;
    /**
     * 点击搜索图标,或者在输入框按回车,或者当前输入框文本值发生变化时的回调。<br>
     * 签名：`function(value: any) => void`<br>
     * {any} value 当前搜索框的文本值。
     */
    onSearch?: (value: any) => void;
    /**
     * 当搜索框值变化时，调用该回调动态设置下拉列表内容,和popItems互斥，两者不能同时设置<br>
     * 签名：`function(value: any) => Array`<br>
     * value 当前搜索输入框的value值<br>
     * returns {Array}  返回下拉列表数据，数据格式和popItems一致
     */
    onSuggest?: (value: any) => any[];
    /**
     * popUp中每项数据被点击之后的回调函数
     */
    onItemClick?: (value: any, obj: object) => void;
    /**
     * 点击清除图标的回调。<br>
     * 签名：`function(value: any) => void`<br>
     * function(value: any) => void
     */
    onClear?: (value: any) => void;
    /**
     * 默认：`false`<br>
     * to show the popup when the user set as true or hide the popup .
     */
    showPopUp?: boolean;
    /**
     * Object that defines the lazy search options.
     * If this property is defined, then the records that are being shown in the search suggestion drop down will be paged, and lazy loaded through the defined callback method.
     * User need to provide only a set of 10 records at a time.
     * The data structure is as :
     * `totalRecords`: The total number of search results for the searched string
     * `onLoadRecords`: The callback that will load next set of the data to be displayed in the popup. This method will have the index of the first element in the view port as the parameter.
     */
    lazySearch?: object;
    virtualScroll?: boolean;
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
     * SearchInput 组件的静态属性 defaultValidator 提供了下列默认校验函数：min, max, range, number, email, digit, url, alpha, regex, postfix, ipv4, ipv6, creditCard, equalTo, notEqualTo, minLength, maxLength, rangeLength。例如，如果希望用户输入的最小值是 5, 可以这样使用 <TextField validator={TextField.defaultValidator.min(5)}/>
     */
    validator?: (value: string, id?: string, type?: string) => {
        result: boolean;
        message?: string | Element;
    };
    /**
     *
     */
    inputProps?: React.InputHTMLAttributes<HTMLElement>;
    /**
     * 搜索框值更改触发事件。<br>
     * 签名：`function(value: string) => void`<br>
     * function(value: string) => void
     */
    onChange?: (value: string) => void;
    /**
     *  销毁popUp之后的回调函数
     */
    onClosePopup?: () => void;
    /**
     * 当搜索框值失焦时，调用该属性返回搜索框的文本值<br>
     * 签名：`function(value: string) => void`<br>
     * value: 当前搜索框的文本值
     */
    onBlur?: (value: string) => void;
    /**
     * 默认：`true`<br>
     *  在有失焦事件的情况下，失焦时是否去除字符串两端空白字符。
     */
    isBlurTrim?: boolean;
    /**
     * 搜索框聚焦事件
     */
    onFocus?: () => void;
    /**
     * 默认：`9999`<br>
     *  zindex value for popup
     */
    zindex?: string;
    /**
     * 默认：`false`<br>
     * 控制自定义提示信息的展示，当 value 长度超过文本框长度时展示
     */
    showTip?: boolean;
    /**
     * 默认：`true`<br>
     * 是否显示clearButton
     */
    clearButton?: boolean;
    /**
   * 默认：`false`<br>
   * 是否显示加载图标
   */
    isLoading?: boolean;
    [restProps: string]: any;
    /**
     * 默认：`div`<br>
     * 设置提示的类型
     */
    hintType?: 'div' | 'tip';
    /**
     * 默认：`true`<br>
     * 设置提示的类型
     */
    isAllowSpaceBar?: boolean;
    /**
   * 默认：`true`<br>
   * 是否开启自动搜索
   */
    autoSearch?: boolean;
    isCharacterAllowed?: any;
};
export interface SearchInputState {
    placeholderVal: string;
    value: string;
    popItems: popItemsType[];
    isShow: boolean;
    maxLengthInput: number;
    position: string;
    editing: boolean;
    error: boolean;
    displayTip: boolean;
    popupIndex: number | null;
    rectWidth: number | null;
    tipText: any;
    hasFocus: boolean;
    hasValue: boolean;
    hasError: boolean;
    selectedIndex: number;
}
export {};
