import { FC } from 'react';
type MinimizeProps = {
    isMinimized: boolean;
    onClick: () => void;
};
declare const MinimizeIcon: FC<MinimizeProps>;
export default MinimizeIcon;
