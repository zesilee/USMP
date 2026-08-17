import React, { Component } from 'react';
import { SearchInputProps, SearchInputState } from './type';
export default class SearchInput extends Component<SearchInputProps> {
    static defaultProps: SearchInputProps;
    validateType: string;
    state: SearchInputState;
    indexCount: number;
    allDisabled: boolean;
    originalSelectedValue: any;
    tip: any;
    pop: any;
    dom: HTMLDivElement;
    input: HTMLInputElement;
    compositionStatus: string;
    static contextType: React.Context<import("react-intl").IntlShape>;
    formatMessage: any;
    constructor(props: SearchInputProps, context: any);
    id: string;
    bindEvents: () => void;
    isAllDisabled: () => void;
    handleScroll: (e: MouseEvent) => void;
    onCompositionStart: () => void;
    onCompositionEnd: (e: any) => void;
    componentWillUnmount(): void;
    checkBrowserAndScroll: (element: any) => void;
    scrollSelectedIntoView: () => void;
    componentDidUpdate(): void;
    initPosition(): string;
    /**
     * 计算下拉框的展开方向
     * @returns {boolean}
     */
    getPopPosition(): boolean;
    handleSearch: () => void;
    handleClear: () => void;
    handleUpAndDownKey: (event: any) => void;
    escapeRegExp(str: any): any;
    validate: () => boolean;
    handleChange: (event: any) => void;
    handleBlur: (e: any) => void;
    getValue(): string;
    getIndexByValue(value: any, data: any): any;
    handleItemClick: (obj: any) => void;
    handleInputFocus: () => void;
    handleInputClick: () => void;
    handleInputBlur: () => void;
    componentWillReceiveProps(props: any): void;
    handleMouseOver: () => void;
    handleMouseOut: (e: any) => void;
    isChild: (element: any) => boolean;
    handleKeyClear: (event: any) => void;
    handleUpAndDownKeys: (event: any) => void;
    focus(): void;
    getInputTitle(title: any, placeholder: any, showTip: any): any;
    render(): React.JSX.Element;
}
