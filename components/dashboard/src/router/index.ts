import Vue from 'vue'
import VueRouter from 'vue-router'
Vue.use(VueRouter)

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import(/* webpackChunkName: "home" */ '../views/overviews/Dashboard.vue')
  },
  {
    path: '/overview/packages',
    name: 'packages',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "packages" */ '../views/overviews/Package.vue')
  },
  {
    path: '/package/:vendor/:package',
    name: 'packageDetail',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "packages" */ '../views/packages/PackageDetail.vue')
  },
  {
    path: '/system/logs',
    name: 'logs',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "logs" */ '../views/systems/Log.vue')
  },
  {
    path: '/system/preferences',
    name: 'preferences',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "preferences" */ '../views/systems/Preference.vue')
  },
  {
    path: '/system/monitor',
    name: 'monitor',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "logs" */ '../views/systems/Monitor.vue')
  },
  {
    path: '/extensions/webhooks',
    name: 'Webhooks',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "preferences" */ '../views/extensions/Webhooks.vue')
  },
  {
    path: '/extensions/apps',
    name: 'Apps',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "preferences" */ '../views/extensions/Apps.vue')
  },
  {
    path: '/experiment/dataset',
    name: 'Dataset',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "preferences" */ '../views/experiment/Dataset.vue')
  },
  {
    path: '/experiment/new',
    name: 'NewExperiment',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "preferences" */ '../views/experiment/Experiment.vue')
  },
  {
    path: '/experiment/experiment/:eid',
    name: 'ExperimentDetail',
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "preferences" */ '../views/experiment/ExperimentDetail.vue')
  },
  {
    path: "**",
    name: "http404",
    component: () => import(/* webpackChunkName: "http404" */ '../views/errors/NotFound.vue')
  }
]

const router = new VueRouter({
  routes
})

export default router
