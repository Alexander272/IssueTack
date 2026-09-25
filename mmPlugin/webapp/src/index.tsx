import './index.css';

import {setStore} from './api';
import type {PluginRegistry, PluginStore} from './types';
import ChannelHeaderIcon from './components/ChannelHeaderIcon';

class IssuetrackPlugin {
    initialize(registry: PluginRegistry, store: PluginStore): void {
        setStore(store);
        registry.registerChannelHeaderIcon({component: ChannelHeaderIcon});
    }
}

window.registerPlugin('issuetrack', new IssuetrackPlugin());