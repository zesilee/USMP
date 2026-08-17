import React from 'react';
import { ConfigProviderProps } from './type';
export declare const ConfigContext: React.Context<ConfigProviderProps>;
declare function ConfigProvider(baseProps: ConfigProviderProps): React.JSX.Element;
declare namespace ConfigProvider {
    var ConfigContext: React.Context<ConfigProviderProps>;
}
export default ConfigProvider;
