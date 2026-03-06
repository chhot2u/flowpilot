export namespace models {
	
	export class ProxyConfig {
	    server: string;
	    protocol?: string;
	    username?: string;
	    password?: string;
	    geo?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.protocol = source["protocol"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.geo = source["geo"];
	    }
	}
	export class TaskStep {
	    action: string;
	    selector?: string;
	    value?: string;
	    timeout?: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.selector = source["selector"];
	        this.value = source["value"];
	        this.timeout = source["timeout"];
	    }
	}
	export class BatchTaskInput {
	    name: string;
	    url: string;
	    steps: TaskStep[];
	    proxy: ProxyConfig;
	    priority: number;
	
	    static createFrom(source: any = {}) {
	        return new BatchTaskInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.steps = this.convertValues(source["steps"], TaskStep);
	        this.proxy = this.convertValues(source["proxy"], ProxyConfig);
	        this.priority = source["priority"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class AdvancedBatchInput {
	    flowId: string;
	    urls: string[];
	    namingTemplate: string;
	    priority: number;
	    proxy: ProxyConfig;
	    tags?: string[];
	    autoStart: boolean;

	    static createFrom(source: any = {}) {
	        return new AdvancedBatchInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.flowId = source["flowId"];
	        this.urls = source["urls"] || [];
	        this.namingTemplate = source["namingTemplate"];
	        this.priority = source["priority"];
	        this.proxy = this.convertValues(source["proxy"], ProxyConfig);
	        this.tags = source["tags"];
	        this.autoStart = source["autoStart"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class RecordedStep {
	    index: number;
	    action: string;
	    selector?: string;
	    value?: string;
	    timeout?: number;
	    snapshotId?: string;
	    // Go type: time
	    timestamp: any;

	    static createFrom(source: any = {}) {
	        return new RecordedStep(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.action = source["action"];
	        this.selector = source["selector"];
	        this.value = source["value"];
	        this.timeout = source["timeout"];
	        this.snapshotId = source["snapshotId"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class RecordedFlow {
	    id: string;
	    name: string;
	    description?: string;
	    steps: RecordedStep[];
	    originUrl: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;

	    static createFrom(source: any = {}) {
	        return new RecordedFlow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.steps = this.convertValues(source["steps"], RecordedStep);
	        this.originUrl = source["originUrl"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class DOMSnapshot {
	    id: string;
	    flowId: string;
	    stepIndex: number;
	    html: string;
	    screenshotPath: string;
	    url: string;
	    // Go type: time
	    capturedAt: any;

	    static createFrom(source: any = {}) {
	        return new DOMSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.flowId = source["flowId"];
	        this.stepIndex = source["stepIndex"];
	        this.html = source["html"];
	        this.screenshotPath = source["screenshotPath"];
	        this.url = source["url"];
	        this.capturedAt = this.convertValues(source["capturedAt"], null);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class BatchGroup {
	    id: string;
	    flowId: string;
	    taskIds: string[];
	    total: number;
	    name: string;

	    static createFrom(source: any = {}) {
	        return new BatchGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.flowId = source["flowId"];
	        this.taskIds = source["taskIds"] || [];
	        this.total = source["total"];
	        this.name = source["name"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class BatchProgress {
	    batchId: string;
	    total: number;
	    pending: number;
	    queued: number;
	    running: number;
	    completed: number;
	    failed: number;
	    cancelled: number;

	    static createFrom(source: any = {}) {
	        return new BatchProgress(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.batchId = source["batchId"];
	        this.total = source["total"];
	        this.pending = source["pending"];
	        this.queued = source["queued"];
	        this.running = source["running"];
	        this.completed = source["completed"];
	        this.failed = source["failed"];
	        this.cancelled = source["cancelled"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class QueueMetrics {
	    running: number;
	    queued: number;
	    pending: number;
	    totalSubmitted: number;
	    totalCompleted: number;
	    totalFailed: number;

	    static createFrom(source: any = {}) {
	        return new QueueMetrics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.queued = source["queued"];
	        this.pending = source["pending"];
	        this.totalSubmitted = source["totalSubmitted"];
	        this.totalCompleted = source["totalCompleted"];
	        this.totalFailed = source["totalFailed"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LogEntry {
	    // Go type: time
	    timestamp: any;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Proxy {
	    id: string;
	    server: string;
	    protocol: string;
	    username?: string;
	    password?: string;
	    geo?: string;
	    status: string;
	    latency: number;
	    successRate: number;
	    totalUsed: number;
	    // Go type: time
	    lastChecked?: any;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Proxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.server = source["server"];
	        this.protocol = source["protocol"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.geo = source["geo"];
	        this.status = source["status"];
	        this.latency = source["latency"];
	        this.successRate = source["successRate"];
	        this.totalUsed = source["totalUsed"];
	        this.lastChecked = this.convertValues(source["lastChecked"], null);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TaskResult {
	    taskId: string;
	    success: boolean;
	    extractedData?: Record<string, string>;
	    screenshots?: string[];
	    logs: LogEntry[];
	    duration: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.success = source["success"];
	        this.extractedData = source["extractedData"];
	        this.screenshots = source["screenshots"];
	        this.logs = this.convertValues(source["logs"], LogEntry);
	        this.duration = source["duration"];
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Task {
	    id: string;
	    name: string;
	    url: string;
	    steps: TaskStep[];
	    proxy: ProxyConfig;
	    priority: number;
	    status: string;
	    retryCount: number;
	    maxRetries: number;
	    timeout?: number;
	    error?: string;
	    result?: TaskResult;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    startedAt?: any;
	    // Go type: time
	    completedAt?: any;
	    tags?: string[];
	    batchId?: string;
	    flowId?: string;
	    headless?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.steps = this.convertValues(source["steps"], TaskStep);
	        this.proxy = this.convertValues(source["proxy"], ProxyConfig);
	        this.priority = source["priority"];
	        this.status = source["status"];
	        this.retryCount = source["retryCount"];
	        this.maxRetries = source["maxRetries"];
	        this.timeout = source["timeout"];
	        this.error = source["error"];
	        this.result = this.convertValues(source["result"], TaskResult);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.tags = source["tags"];
	        this.batchId = source["batchId"];
	        this.flowId = source["flowId"];
	        this.headless = source["headless"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}
