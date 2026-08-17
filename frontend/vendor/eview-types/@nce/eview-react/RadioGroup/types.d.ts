export type RadioGroupProps = {
    /**
     * 设置RadioGroup的数据：let radios1=[{value?:1,text?:"MALE"},{value?:2,text?:"222"},{value?:3,text?:"333"}]
     * value为数据库存取值，text为文本展示值
     */
    data?: any;
    /**
     * 设置radioGroup的文本子名称
     */
    label?: string;
    /**
     * 设置RadioGroup的id
     */
    id?: string;
    /**
     * 通过自定义class来改变RadioGroup的整体样式
     * 例如：margin、padding
     */
    className?: string;
    /**
     * 通过自定义style来改变RadioGroup的整体样式
     * 例如：margin、padding
     */
    style?: object;
    /**
     * 通过添加class的方式控制label的样式，可用来调节单选组和单选组文本名字之间的间距
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式控制label的样式，可用来调节单选组和单选组文本名字之间的间距
     */
    labelStyle?: object;
    /**
     * 默认：`before`<br>
     * 设置radioGroup文本值对于radioGroup的位置
     * before代表文本值在radioGroup的左边，after代表文本值在radioGroup的右边
     */
    labelPosition?: 'before' | 'after';
    /**
     * 设置选中的value值
     */
    value?: any;
    /**
     * 默认：`false`<br>
     * 设置组件是否为必填项
     */
    required?: boolean;
    /**
     * 默认：`false`<br>
     * 设置组件的灰化状态
     */
    disabled?: boolean;
    /**
     * 设置RadioGroup的onClick事件，会传递参数为：<br>
     * 签名：`function(value: any, oldValue: any, event: object) => void`<br>
     * {any} value 当前选中值<br>
     * {any} oldValue 上次选中值<br>
     * {object} event 原生dom的点击事件
     */
    onChange?: (oldValue: any, value: any, event: object) => void;
    /**
     * Specifies radio elements alignment in multiple columns and rows.
     * if rows props not mentioned it will be in normal strucure
     *
     */
    rows?: string;
    /**
     * Specifies radio elements alignment in multiple rows.
     * if rowSpacing props not mentioned it will be in normal strucure
     *
     */
    rowSpacing?: string;
    /**
     * Specifies radio elements alignment in multiple columns.
     * if colSpacing props not mentioned it will be in normal strucure
     *
     */
    colSpacing?: string;
    title?: string;
    /**
     * 设置为 受控组件
     */
    isControlled?: boolean;
    /**
     * 默认：`horizontal`<br>
     * 设置横向纵向展示
     */
    type?: 'vertical' | 'horizontal';
    /**
     * 显示提示信息的样式类型
     */
    hintType?: '' | 'div' | 'tip';
};
export interface RadioGroupState {
}
