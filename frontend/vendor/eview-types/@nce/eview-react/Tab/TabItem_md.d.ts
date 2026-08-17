/**
 * TabItem的ts注释上的部分属性(index,className等)是Tab传递过来的，其本身是不具有的。所以重新创建一份文件暴露，来生成对应的md
 */
import React from 'react';
export type TabItemProps = {
    /**
     * 设置TabItem的id
     */
    id?: string;
    /**
     * tab的标题内容
     */
    title?: string;
    /**
     * 默认：`false`<br>
     * 设置标签是否可关闭
     */
    closable?: boolean;
    /**
     * 默认：`false`<br>
     * 设置Tab的灰化状态
     */
    disabled?: boolean;
    /**
     * 页签项的图标。并在此处设置图标的绝对路径。
     */
    icon?: string | React.ReactElement;
    /**
     * 默认：`false`<br>
     * 页签项的图标。并在此处设置图标的绝对路径。
     */
    setEditing?: boolean;
    /**
     * 设置页签标题的样式（比如字体大小、颜色）
     */
    tabItemStyle?: React.CSSProperties;
    /**
     * 设置tab标题自定义的内容
     */
    titleExtraContent?: React.ReactElement;
    /**
     * 设置鼠标悬浮在item上的提示内容
     */
    itemTip?: string;
};
