export namespace execute {
	
	export class CommandResult {
	    lapse: number;
	    exitCode: number;
	    stdout: string;
	    stderr: string;
	    error: string;
	    aborted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CommandResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lapse = source["lapse"];
	        this.exitCode = source["exitCode"];
	        this.stdout = source["stdout"];
	        this.stderr = source["stderr"];
	        this.error = source["error"];
	        this.aborted = source["aborted"];
	    }
	}

}

export namespace porter {
	
	export class ImportOptions {
	    settings: boolean;
	    data: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = source["settings"];
	        this.data = source["data"];
	    }
	}
	export class ImportPreview {
	    // Go type: time
	    exportedAt: any;
	    hasSettings: boolean;
	    hasData: boolean;
	    hasDatabase: boolean;
	    hasDrivers: boolean;
	    driverCount: number;
	    driverSize: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exportedAt = this.convertValues(source["exportedAt"], null);
	        this.hasSettings = source["hasSettings"];
	        this.hasData = source["hasData"];
	        this.hasDatabase = source["hasDatabase"];
	        this.hasDrivers = source["hasDrivers"];
	        this.driverCount = source["driverCount"];
	        this.driverSize = source["driverSize"];
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
	export class JobSnapshot {
	    status: status.Status;
	    step: string;
	    progress: number;
	    messages: string[];
	
	    static createFrom(source: any = {}) {
	        return new JobSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.step = source["step"];
	        this.progress = source["progress"];
	        this.messages = source["messages"];
	    }
	}

}

export namespace status {
	
	export enum Status {
	    ABORTED = "aborted",
	    ABORTING = "aborting",
	    COMPLETED = "completed",
	    ERRORED = "errored",
	    FAILED = "failed",
	    PENDING = "pending",
	    RUNNING = "running",
	    SKIPED = "skiped",
	    SPEEDED = "speeded",
	}

}

export namespace storage {
	
	export enum DriverType {
	    DISPLAY = "display",
	    MISCELLANEOUS = "miscellaneous",
	    NETWORK = "network",
	}
	export enum RuleOperator {
	    CONTAIN = "contain",
	    EQUAL = "equal",
	    NOT_CONTAIN = "notContain",
	    NOT_EQUAL = "notEqual",
	    REGEX = "regex",
	}
	export enum RuleSource {
	    CPU = "cpu",
	    DISK = "storage",
	    GPU = "gpu",
	    MEMORY = "memory",
	    MOTHERBOARD = "motherboard",
	    NIC = "nic",
	}
	export enum SuccessAction {
	    FIRMWARE = "firmware",
	    NOTHING = "nothing",
	    REBOOT = "reboot",
	    SHUTDOWN = "shutdown",
	}
	export class AppSetting {
	    create_partition: boolean;
	    set_password: boolean;
	    password: string;
	    parallel_install: boolean;
	    success_action: SuccessAction;
	    success_action_delay: number;
	    filter_miniport_nic: boolean;
	    filter_microsoft_nic: boolean;
	    language: string;
	    driver_download_url: string;
	    auto_check_update: boolean;
	    hide_not_found: boolean;
	    allow_pre_release: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.create_partition = source["create_partition"];
	        this.set_password = source["set_password"];
	        this.password = source["password"];
	        this.parallel_install = source["parallel_install"];
	        this.success_action = source["success_action"];
	        this.success_action_delay = source["success_action_delay"];
	        this.filter_miniport_nic = source["filter_miniport_nic"];
	        this.filter_microsoft_nic = source["filter_microsoft_nic"];
	        this.language = source["language"];
	        this.driver_download_url = source["driver_download_url"];
	        this.auto_check_update = source["auto_check_update"];
	        this.hide_not_found = source["hide_not_found"];
	        this.allow_pre_release = source["allow_pre_release"];
	    }
	}
	export class Driver {
	    id: number;
	    name: string;
	    type: DriverType;
	    path: string;
	    flags: string[];
	    minExeTime: number;
	    allowRtCodes: number[];
	    incompatibles: number[];
	
	    static createFrom(source: any = {}) {
	        return new Driver(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.path = source["path"];
	        this.flags = source["flags"];
	        this.minExeTime = source["minExeTime"];
	        this.allowRtCodes = source["allowRtCodes"];
	        this.incompatibles = source["incompatibles"];
	    }
	}
	export class DriverGroup {
	    id: number;
	    name: string;
	    type: DriverType;
	    mutuallyExclusive: boolean;
	    drivers: Driver[];
	
	    static createFrom(source: any = {}) {
	        return new DriverGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.mutuallyExclusive = source["mutuallyExclusive"];
	        this.drivers = this.convertValues(source["drivers"], Driver);
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
	export class Rule {
	    source: RuleSource;
	    operator: RuleOperator;
	    is_case_sensitive: boolean;
	    should_hit_all: boolean;
	    values: string[];
	
	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.operator = source["operator"];
	        this.is_case_sensitive = source["is_case_sensitive"];
	        this.should_hit_all = source["should_hit_all"];
	        this.values = source["values"];
	    }
	}
	export class RuleSet {
	    id: number;
	    name: string;
	    rules: Rule[];
	    should_hit_all: boolean;
	    driver_group_ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new RuleSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.rules = this.convertValues(source["rules"], Rule);
	        this.should_hit_all = source["should_hit_all"];
	        this.driver_group_ids = source["driver_group_ids"];
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

export namespace sysinfo {
	
	export class ResolvedHardware {
	    cpu: string[];
	    gpu: string[];
	    memory: string[];
	    motherboard: string[];
	    nic: string[];
	    storage: string[];
	
	    static createFrom(source: any = {}) {
	        return new ResolvedHardware(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.gpu = source["gpu"];
	        this.memory = source["memory"];
	        this.motherboard = source["motherboard"];
	        this.nic = source["nic"];
	        this.storage = source["storage"];
	    }
	}

}

export namespace update {
	
	export class UpdateCheckResult {
	    hasUpdate: boolean;
	    latestVersion: string;
	    downloadUrl: string;
	    downloadUrlBundled: string;
	    releaseNotes: string;
	    releaseAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasUpdate = source["hasUpdate"];
	        this.latestVersion = source["latestVersion"];
	        this.downloadUrl = source["downloadUrl"];
	        this.downloadUrlBundled = source["downloadUrlBundled"];
	        this.releaseNotes = source["releaseNotes"];
	        this.releaseAt = source["releaseAt"];
	    }
	}

}

