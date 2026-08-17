import React from 'react';
export type LoadingProps = {
    /**
     * 外层容器id
     */
    id?: string;
    /**
     * 默认：`false`<br>
     * 自定义弹框是否显示，默认值为false（关闭）
     * IsOpen = {true}
     */
    isOpen?: boolean;
    /**
     * 默认：`global`<br>
     * 指定元素的类型
     */
    type?: 'global' | 'local' | 'micro';
    /**
     * 指定图标路径，将显示为加载/刷新图标
     * 这是必填字段
     */
    iconUrl?: string;
    /**
     * 指定图标/流程说明
     */
    desc?: string;
    /**
     * 为标签设置样式
     */
    textClassName?: string;
    /**
     * 设置元素的样式
     */
    style?: React.CSSProperties;
    className?: string;
};
declare const Loading: React.FC<LoadingProps>;
export default Loading;
