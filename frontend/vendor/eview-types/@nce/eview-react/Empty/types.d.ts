import { ReactNode, CSSProperties } from 'react';
export type EmptyProps = {
    /**
     * 设置组件最外层的内联样式
     */
    style?: CSSProperties;
    /**
     * 自定义的类名
     */
    className?: string;
    /**
     * 描述文本
     */
    description?: ReactNode;
    /**
     * 自定义icon
     */
    icon?: ReactNode;
    /**
     * 自定义图片
     */
    imgSrc?: string;
    /**
     * 设置两种默认类型
     */
    type?: 'success' | 'fail';
};
