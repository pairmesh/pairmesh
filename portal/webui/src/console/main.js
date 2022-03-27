// Copyright 2022 PairMesh, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import Vue from 'vue'
import Console from './app'
import Overview from '@/console/components/overview'
import Networks from '@/console/components/networks'
import Keys from '@/console/components/keys'
import Settings from '@/console/components/settings'
import Member from '@/console/components/member'
import '@/plugins/element.js'
import '@/console/components/flexbox'
import '@/plugins/vue-router'
import VueRouter from "vue-router";

const router = new VueRouter({
    mode: 'history',
    routes: [
        {
            path: '/console/overview',
            component: Overview,
        },
        {
            path: '/console/networks',
            component: Networks,
        },
        {
            path: '/console/network/:network_id',
            component: Member,
        },
        {
            path: '/console/keys',
            component: Keys,
        },
        {
            path: '/console/settings',
            component: Settings,
        }
    ],
})

new Vue({
    router,
    el: '#app',
    render: h => h(Console)
});
