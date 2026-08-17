import React from 'react';
export interface I18nProviderProps {
    locale?: string;
    messages?: object;
    children?: any;
}
declare const I18nProvider: (props: I18nProviderProps) => React.JSX.Element;
export default I18nProvider;
