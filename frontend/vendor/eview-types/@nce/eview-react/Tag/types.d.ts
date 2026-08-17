import React from "react";
export type TagProps = {
    /**
     * 自定义class改变tag的整体样式，作用于最外层div
     */
    className?: string;
    /**
     * 自定义id改变tag的整体样式，作用于最外层div
     */
    id?: any;
    /**
     * 默认：`true`<br> 小尺寸圆角0.75rem，大尺寸圆角1rem
     * 是否圆角
     */
    round?: boolean;
    /**
     * 默认：`solid`<br>
     * 填充模式内部填充/外部填充<br>
     * 填充模式当为 solid 时，设置的 color 即为背景色，文字与边框颜色均为白色；当为 outline 时，设置的 color 即为文字与边框颜色，背景色为白色
     */
    fill?: 'solid' | 'outline';
    children?: React.ReactNode;
    /**
     * 默认：`default`<br>
     * 填充颜色
     */
    color?: (string & {}) | 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'caution';
    /**
     * 标签点击事件
     */
    onClick?: (e: React.MouseEvent<HTMLSpanElement, MouseEvent>) => void;
    /**
     * 自定义的css变量样式
     */
    style?: React.CSSProperties;
    /**
     * 默认：`normal`<br>
     * 设置组件的大小，'normal'为小尺寸, 'large'为大尺寸型
     */
    size?: 'small' | 'normal' | 'large';
    /**
  * 默认：`false`<br>
  * 是否属于信息标签
  */
    isMessageTag?: boolean;
    /**
  * 默认：`false`<br>
  * 信息标签是否有图标
  */
    hasIcon?: boolean;
    /**
     * 设置 Icon 的附加属性<br/>
     * 属性：<br/>
     *  iconUrl 通过传入路径设置自定义icon iconUrl= 'image/huawei.png'<br/>
     * hoverColor 定制鼠标悬浮时icon的颜色 <br/>
     *  style 定制icon行内样式<br/>
     * className 自定义icon样式
     */
    tagIconProps?: IconProps;
    /**定制图标标识名，标识名与icon名称相同；<br/>
     * 例如： iconName='arrow_down'
     *  */
    iconName?: string;
};
type IconProps = {
    iconUrl?: string;
    hoverColor?: string;
    style?: React.CSSProperties;
    className?: string;
};
export type CssDefaultProps = {
    /**
     * 当 fill=solid 时，默认值为 #ffffff；当 fill=outline 时，默认值为 color 属性对应的颜色<br>
     * 文字颜色
     */
    color?: string;
    /**
     * 当 fill=solid 时，默认值为 color 属性对应的颜色；当 fill=outline 时，默认值为 #ffffff<br>
     * 背景颜色
     */
    background?: string;
    /**
     * color 属性对应的颜色<br>
     * 边框颜色
     */
    borderColor?: string;
    /**
     * 默认：`'2px'`<br>
     * round=false 时的圆角大小
     */
    borderRadius?: string;
    /**
     * 默认：`'1px solid borderColor'`<br>
     * 边框设置
     */
    border?: string;
};
export {};
