import React from 'react';
export interface LoadingWindowProps {
    /**
     * "将内容设置到窗口.
     */
    content?: React.ReactNode;
    /**
     * 窗口宽度.
     */
    width?: string;
    /**
     * 窗口高度.
     */
    height?: string;
    /**
     * 用户可以将窗口设置为模式或非模式。默认值为模态。如果用户希望窗口设置为非模态，则模态选项必须设置为false。
     */
    modal?: boolean;
    children?: never[];
    /**
     * 设置元素的样式
     */
    style?: React.CSSProperties;
}
declare const LoadingWindow: (props: LoadingWindowProps) => React.JSX.Element;
export default LoadingWindow;
