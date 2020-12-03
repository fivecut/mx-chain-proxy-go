import {Err} from "@elrondnetwork/erdjs/out";
import {TestPhase, TestStatus, TestSuite} from "../../resultClasses";
import {displayErrResponse} from "../../htmlRenders";
import {CommonHandler} from "../commonHandler";

/*
    NodeV1_0Handler is the class that will handle the node API calls for the API v1.0
 */
export class NodeV1_0Handler {
    commonHandler: CommonHandler;

    constructor(commonHandler: CommonHandler) {
        this.commonHandler = commonHandler;
    }

    async handleHeartbeat(): Promise<TestSuite> {
        let testPhases = new Array<TestPhase>();

        let url = this.commonHandler.proxyURL + "/node/heartbeatstatus";
        try {
            let response = await this.commonHandler.httpRequestHandler.doGetRequest(url)
            testPhases.push(this.commonHandler.runBasicTestPhaseOk(response, 200))

            return new TestSuite("v1.0", testPhases, TestStatus.SUCCESSFUL, response)

        } catch (error) {
            displayErrResponse("LoadHeartBeatOutput", url, Err.html(error))
            return new TestSuite("v1.0", testPhases, TestStatus.UNSUCCESSFUL, null)
        }
    }
}
