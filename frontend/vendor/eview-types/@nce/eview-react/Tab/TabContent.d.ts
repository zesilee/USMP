import React, { Component } from 'react';
export interface TabContentProps {
    className?: string;
    style?: React.CSSProperties;
    lazyLoad?: boolean;
    selectedIndex?: number;
    itemStatus?: any[];
    contentChildren?: any;
    onKeyDown?: (e: React.KeyboardEvent) => void;
    position?: 'top' | 'bottom' | 'left' | 'right';
    isUpdateContent?: boolean;
}
export default class TabContent extends Component<TabContentProps> {
    container: HTMLDivElement;
    getTabContent(): any;
    shouldComponentUpdate(nextProps: any, nextState: any): boolean;
    render(): React.JSX.Element;
}
