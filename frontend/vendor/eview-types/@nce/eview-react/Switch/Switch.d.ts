import React from 'react';
export type labelPosition = 'before' | 'after';
export type SwitchProps = {
    /**
     * 切换框根节点的id。
     */
    id?: string;
    /**
     * 切换框最外层div的样式类名。
     */
    className?: string;
    /**
     * 覆盖切换框最外层div的行内样式。
     */
    style?: React.CSSProperties;
    /**
     * 通过添加class的方式控制label的样式,以及展示文本与组件之间的间距
     */
    labelClassName?: string;
    /**
     * 通过添加style的方式控制label的样式,以及展示文本与组件之间的间距
     */
    labelStyle?: React.CSSProperties;
    /**
     * 切换框不同状态下的值集合，默认圆圈在左边时表示关闭状态，圆圈在右边时表示开启状态，
     * 比如["33","44"],第一个值表示关闭状态下的值，第二个值表示开启状态下的值，
     * 这个属性可选，推荐设置默认值。
     */
    data?: any[];
    /**
     * 设置组件的展示文本值
     */
    label?: string;
    /**
     * 默认：`before`<br>
     * 设置组件的文本值在组件图标的左侧还是右侧
     * before代表文本在组件的左边，after表示文本在组件的右边
     */
    labelPosition?: labelPosition;
    /**
     * 默认：`false`<br>
     * 设置开启切换框，默认关闭。
     */
    toggled?: boolean;
    /**
     * 默认：`false`<br>
     * 设置禁用切换框，默认启用。
     */
    disabled?: boolean;
    /**
     * 点击切换框时的回调。使用时需传 data。<br>
     * 签名：`(value: any) => void`<br>
     *  {any} value 点击文本切换框时当前状态值
     */
    onToggle?: (data: any) => void;
    fieldClassName?: string;
    fieldStyle?: React.CSSProperties;
    /**
     * 默认：`false`<br>
     * 设置组件是否需要必填
     */
    required?: boolean;
    /**
     *
     *   选中时的内容
     */
    taggledChildren?: string | React.ReactNode;
    /**
     *
     *   非选中时的内容
     */
    unTaggledChildren?: string | React.ReactNode;
    /**
   * @defaultValue false
   * @zh  允许向上冒泡
   */
    allowPropagation?: boolean;
    /**
     * 是否开启外部控制状态 ，例如：点击开关后需进行二次确认再为组件设置值
     */
    isControlToggled?: boolean;
};
declare const _default: React.ForwardRefExoticComponent<SwitchProps & React.RefAttributes<any>>;
export default _default;
