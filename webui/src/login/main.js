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
import App from './app.vue'
import '@/plugins/element.js'
import '@/plugins/vue-router'
import VueRouter from "vue-router";
import Login from "@/login/components/login";
import AuthCallback from "@/login/components/auth_callback";
import AuthRedirect from "@/login/components/auth_redirect";

const router = new VueRouter({
    mode: 'history',
    routes: [
        {
            path: '/login',
            component: Login,
        },
        {
            path: '/login/auth/callback/github',
            component: AuthCallback,
        },
        {
            path: '/login/auth/redirect',
            component: AuthRedirect,
        }
    ],
})

new Vue({
    router,
    el: '#app',
    render: h => h(App)
});
