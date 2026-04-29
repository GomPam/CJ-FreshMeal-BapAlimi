export namespace main {
	
	export class ConfigTarget {
	    id: string;
	    name: string;
	    type: string;
	    group: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.group = source["group"];
	    }
	}
	export class AppConfig {
	    storeIdx: string;
	    storeName: string;
	    targets: ConfigTarget[];
	    times: string[];
	    windowX?: number;
	    windowY?: number;
	    hasWindowPos?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storeIdx = source["storeIdx"];
	        this.storeName = source["storeName"];
	        this.targets = this.convertValues(source["targets"], ConfigTarget);
	        this.times = source["times"];
	        this.windowX = source["windowX"];
	        this.windowY = source["windowY"];
	        this.hasWindowPos = source["hasWindowPos"];
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
	
	export class DeviceCode {
	    UserCode: string;
	    VerificationURL: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceCode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UserCode = source["UserCode"];
	        this.VerificationURL = source["VerificationURL"];
	    }
	}
	export class LogEntry {
	    time: string;
	    level: string;
	    action: string;
	    target: string;
	    status: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.action = source["action"];
	        this.target = source["target"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class Schedule {
	    targets: ConfigTarget[];
	    times: string[];
	
	    static createFrom(source: any = {}) {
	        return new Schedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targets = this.convertValues(source["targets"], ConfigTarget);
	        this.times = source["times"];
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
	export class SendTarget {
	    ID: string;
	    Name: string;
	    Type: string;
	    Group: string;
	
	    static createFrom(source: any = {}) {
	        return new SendTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Type = source["Type"];
	        this.Group = source["Group"];
	    }
	}
	export class StoreItem {
	    idx: string;
	    name: string;
	    address: string;
	    workStatus: string;
	    workTime: string;
	    hldTxt: string;
	
	    static createFrom(source: any = {}) {
	        return new StoreItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.idx = source["idx"];
	        this.name = source["name"];
	        this.address = source["address"];
	        this.workStatus = source["workStatus"];
	        this.workTime = source["workTime"];
	        this.hldTxt = source["hldTxt"];
	    }
	}
	export class StoreSearchResult {
	    totalCount: number;
	    storeList: StoreItem[];
	
	    static createFrom(source: any = {}) {
	        return new StoreSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalCount = source["totalCount"];
	        this.storeList = this.convertValues(source["storeList"], StoreItem);
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
	export class UpdateInfo {
	    available: boolean;
	    currentVer: string;
	    latestVer: string;
	    releaseUrl: string;
	    downloadUrl: string;
	    releaseNote: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.currentVer = source["currentVer"];
	        this.latestVer = source["latestVer"];
	        this.releaseUrl = source["releaseUrl"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseNote = source["releaseNote"];
	    }
	}

}

