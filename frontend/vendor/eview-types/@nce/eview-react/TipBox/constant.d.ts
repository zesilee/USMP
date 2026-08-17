export declare const HOVER = "hover";
export declare const FOCUS = "focus";
export declare const CLICK = "click";
export type Trigger = typeof HOVER | typeof FOCUS | typeof CLICK;
export type TriggerEventName = 'onMouseEnter' | 'onMouseLeave' | 'onFocus' | 'onBlur' | 'onClick';
export declare const TriggerEventMapping: {
    [k: string]: {
        in: TriggerEventName;
        out?: TriggerEventName;
    };
};
export declare const TooltipHostPositionChangedEventName = "TooltipHostPositionChanged";
export type Direction = 'topLeft' | 'top' | 'topRight' | 'rightTop' | 'right' | 'rightBottom' | 'bottomRight' | 'bottom' | 'bottomLeft' | 'leftBottom' | 'left' | 'leftTop';
