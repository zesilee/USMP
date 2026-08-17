import React from 'react';
export interface LoadingBodyProps {
    /**
     * 指定图标路径，将显示为加载/刷新图标
     * 这是必填字段
     */
    iconUrl?: string;
    /**
     * 指定图标/流程说明
     */
    text?: string;
    /**
     * 为标签设置样式
     */
    className: string;
    /**
     * 指定元素的位置
     */
    elementClassName?: string;
    children?: never[];
}
declare const LoadingBody: (props: LoadingBodyProps) => React.JSX.Element;
export default LoadingBody;
