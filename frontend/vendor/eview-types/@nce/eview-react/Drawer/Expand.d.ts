import React, { Component } from 'react';
export interface ExpandProps {
    /**
     * 动画持续时常
     */
    duration?: number;
    /**
     * 进入动画播放延迟时间
     */
    delay?: number;
    /**
     * 控制DOM元素显示/隐藏
     */
    show?: boolean;
    /**
     * 子元素
     */
    children?: React.ReactNode;
    width?: number;
    height?: number;
    placement: string;
}
export default class Expand extends Component<ExpandProps> {
    static defaultProps: {
        show: boolean;
        duration: number;
    };
    componentWillReceiveProps(props: any): void;
    state: {
        show: boolean;
    };
    forceBrowserRepaint: (node: any) => void;
    render(): React.JSX.Element;
}
