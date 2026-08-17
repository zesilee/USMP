import React from 'react';
type leftIconProp = {
    leftHoverIcon?: string;
    leftDisabledIcon?: string;
    leftIconDisabledClass?: string;
    leftIconClass?: string;
};
type rightIconProp = {
    rightHoverIcon?: string;
    rightDisabledIcon?: string;
    rightIconDisabledClass?: string;
    rightIconClass?: string;
};
export type ButtonProps = {
    /**
     * 设置Button的id
     */
    id?: string;
    /**
     * 通过自定义类的方式给Button添加样式
     */
    className?: string;
    /**
     * 通过添加style的方式给Button添加样式
     */
    style?: object;
    /**
     * 默认：`default`<br>
     * 设置Button的使用场景，默认default样式，特殊场景下设置该属性值为primary
     */
    status?: 'default' | 'primary' | 'risk' | 'text';
    /**
     * 默认:`normal`<br>
     * 设置Button的类型
     */
    size?: 'normal' | 'large' | 'small';
    /**
     * 设置组件的类型
     */
    /**
     * 设置Button中的文本
     */
    text?: any;
    /**
     * 在Button中文本的左边设置图标，此时设置图标的绝对路径即可
     */
    leftIcon?: string | React.ReactElement;
    /**
     * 在Button中文本的右边设置图标，此时设置图标的绝对路径即可
     */
    rightIcon?: string | React.ReactElement;
    /**
     * 通过自定义class设置自定义左图标的样式
     */
    lIconClassName?: string;
    /**
     * 通过自定义class设置自定义左图标的样式
     */
    rIconClassName?: string;
    /**
     * 默认：`false`<br>
     * 设置组件是否默认聚焦
     */
    focused?: boolean;
    /**
     * 默认：`false`<br>
     * 设置Button的灰化状态
     */
    disabled?: boolean;
    /**
     * 设置Button的点击事件（支持原生button的所有事件，不再一一列出）<br>
     * 签名：`function(event: object, additionalData: object) => void`<br>
     *  event: 原生 dom 的点击事件<br>
     *  additionalData: Additional Data passing from the upper level component.
     */
    onClick?: (event: object, value: any) => void;
    /**
     * Button键盘按下事件
     */
    onKeyDown?: any;
    /**
     * Button鼠标移出事件
     */
    onMouseLeave?: any;
    /**
     * Button失焦事件
     */
    onBlur?: any;
    /**
     * You can pass additional data from upper level component and get it on clicking the button
     */
    additionalData?: object;
    /**
     * 默认：`never`<br>
     * Show the title on column header.
     */
    tipShow?: 'always' | 'never' | 'overflow';
    /**
     * Show the custom tip for column header otherwise it will use 'text' string
     */
    tipData?: string;
    /**
     * 设置 leftIcon 的附加属性<br>
     * 属性：<br>
     * leftHoverIcon：光标悬停时的 leftIcon<br>
     * leftDisabledIcon：禁用时的 leftIcon<br>
     * leftIconClass: leftIcon 的自定义类<br>
     * leftIconDisabledClass: 禁用时 leftIcon 的自定义类
     */
    leftIconProps?: leftIconProp;
    /**
     * 设置 rightIcon 的附加属性<br>
     * 属性：<br>
     * rightHoverIcon：光标悬停时的 rightIcon<br>
     * rightDisabledIcon：禁用时的 rightIcon<br>
     * rightIconClass: rightIcon 的自定义类<br>
     * rightIconDisabledClass: 禁用时 rightIcon 的自定义类
     */
    rightIconProps?: rightIconProp;
    /**
     * 默认:`title`<br>
     * 设置 tipType 为'title'，提示框显示浏览器默认的标题
     * 设置 tipType 为'tipbox'，弹出 eView-React 提示框
     */
    tipType?: 'title' | 'tipbox';
    /**
     * Button聚焦事件
     */
    onFocus?: (event: object) => void;
    /**
     * 设置leftIcon样式
     */
    lIconStyle?: any;
    /**
     * 设置rightIcon样式
     */
    rIconStyle?: any;
    /**
     * 设置 Blur 开关
     */
    disableBlur?: boolean;
    /**
     * 按钮内容，同 text
     */
    children?: React.ReactNode;
};
export interface ButtonState {
    content: string;
    hoverOnIcon: boolean;
}
export {};
