declare const SMALL_SIZE = "small";
declare const DEFAULT_SIZE = "default";
declare const BEFORE = "before";
declare const AFTER = "after";
type labelPosition = typeof BEFORE | typeof AFTER;
type size = typeof SMALL_SIZE | typeof DEFAULT_SIZE;
export type SegmentedProps = {
    /**
    * 设置组件的id
    */
    id?: string;
    /**
     * 通过添加class的方式设置组件样式，作用于组件最外层div，可设置组件margin padding
     */
    className?: string;
    /**
     * 通过styke的方式设置组件样式，作用于组件最外层div，可设置组件margin padding
     */
    style?: React.CSSProperties;
    /**
     * 通过添加class的方式设置组件选项卡样式
     */
    itemClassName?: string;
    /**
     * 通过添加style的方式设置组件选项卡样式
     */
    itemStyle?: React.CSSProperties;
    /**
     * 通过添加class的方式自定义label的名称的样式
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式自定义label的名称的样式
     */
    labelStyle?: React.CSSProperties;
    /**
     * 默认：`default`<br>
     * 设置组件的类型，默认为大尺寸选项卡,可选`default`,`small`
     */
    type?: size;
    /**
     * 设置组件的数据，例如：[{value:1,text:"100MB"},{value:2,text:"200MB"},{value:3,text:"500MB"},{value:3,text:"800MB"}]
     */
    data?: ItemType[];
    /**
     * 通过value设置组件的选中项
     */
    value?: string | number;
    /**
     * 设置下拉框的名称文字
     */
    label?: string;
    /**
     * 默认：`false`<br>
     * 设置组件是否需要
     */
    required?: boolean;
    /**
     * 默认：`before`<br>
     *设置组件名称文字的位置,可选`before`,`after`
     before 名称文字在组件的左边，
      after 名称文字在组件的右边
     */
    labelPosition?: labelPosition;
    /**
     * 设置组件的onChange事件<br>
     * 签名：`function(value: string I number, event: object) => void`<br>
     * {string | number} value 当前选中的选项值<br>
     * {object} event 原生dom的点击事件
     */
    onChange?: (value: string | number, event: string | number) => void;
    /**
     * 设置tip是否显示<br>
     *  默认：`true`<br>
     */
    isTipShow?: boolean;
    /**
    * 默认：`false`<br>
    * 组件是否禁用
    */
    disable?: boolean;
};
export type ItemType = {
    /**
     * 选项的值
     */
    value: string | number;
    /**
     * 选项显示文本
     */
    text: string | React.ReactElement;
    /**
     * 默认：`false`<br>
     * 选项是否禁用
     */
    disable?: boolean;
    /**
     * 自定义tips，不设置tipsText时，默认显示text
     */
    tipsText?: string | React.ReactElement;
};
export interface SegmentedState {
    checkedIndex: number;
}
export {};
