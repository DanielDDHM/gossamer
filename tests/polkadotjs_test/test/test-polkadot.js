const { describe } = require('mocha');
const { expect } = require('chai');
const { ApiPromise, WsProvider } = require('@polkadot/api');
const { Keyring } = require('@polkadot/keyring');
const fs = require('fs');

const sleep = (delay) => new Promise((resolve) => setTimeout(resolve, delay));

describe('Testing polkadot.js/api calls:', function () {
    let api;
    let done = false;

    before (async function () {
        const wsProvider = new WsProvider('ws://127.0.0.1:8546');
        ApiPromise.create({provider: wsProvider}).then( async (a) => {
            api = a;

            do {
                await sleep(5000);
            } while (!done)
        }).finally( () => process.exit());
    });


    beforeEach ( async function () {
        // this is for handling connection issues to api, if not connected
        //  wait then try again, if still not corrected, close test
        this.timeout(5000);

        if (api == undefined) {
            await sleep(2000);
        }

        if (api == undefined) {
            process.exit(1);
        }
    });

     after(function () {
         done = true;
     });

    describe('api constants', () => {
        it('call api.genesisHash', async function () {
            const genesisHash = await api.genesisHash;
            expect(genesisHash).to.be.not.null;
            expect(genesisHash).to.have.lengthOf(32);
        });

        it('call api.runtimeMetadata', async function () {
            const runtimeMetadata = await api.runtimeMetadata;
            expect(runtimeMetadata).to.be.not.null;
            expect(runtimeMetadata).to.have.property('magicNumber')
        });

        it('call api.runtimeVersion', async function () {
            const runtimeVersion = await api.runtimeVersion;
            expect(runtimeVersion).to.be.not.null;
            expect(runtimeVersion).to.have.property('specName').contains('westend');
            expect(runtimeVersion).to.have.property('apis').lengthOf.above(10)
        });

        it('call api.libraryInfo', async function () {
            const libraryInfo = await api.libraryInfo;
            expect(libraryInfo).to.be.not.null;
            expect(libraryInfo).to.be.equal('@polkadot/api v9.5.1');
        });
    });
    describe('upgrade runtime', () => {
        it('update the runtime using wasm file', async function () {
            const keyring = new Keyring({type: 'sr25519' });
            const aliceKey = keyring.addFromUri('//Alice');

            // Retrieve the runtime to upgrade
            const code = fs.readFileSync('test/node_runtime.compact.wasm').toString('hex');
            const proposal = api.tx.system && api.tx.system.setCode
                ? api.tx.system.setCode(`0x${code}`) // For newer versions of Substrate
                : api.tx.consensus.setCode(`0x${code}`); // For previous versions

            // Perform the actual chain upgrade via the sudo module
            api.tx.sudo
                .sudo(proposal)
                .signAndSend(aliceKey, ({ events = [], status }) => {
                    console.log('Proposal status:', status.type);

                    if (status.isInBlock) {
                        console.error('You have just upgraded your chain');

                        console.log('Included at block hash', status.asInBlock.toHex());
                        console.log('Events:');

                        console.log(JSON.stringify(events.toHuman(), null, 2));
                    } else if (status.isFinalized) {
                        console.log('Finalized block hash', status.asFinalized.toHex());

                        process.exit(0);
                    }
                });
        })
    });
    //TODO: remove skip when rpc.state.queryStorage is fixed (in PR#2505)
    describe.skip('api query', () => {
        it('call api.query.timestamp.now()', async function () {
            const timestamp = await api.query.timestamp.now();
            expect(timestamp).to.be.not.undefined;
        });

        it('call api.query.system.account(ADDR_Alice)', async function () {
            const ADDR_Alice = '5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY';
            const account = await api.query.system.account(ADDR_Alice);
            expect(account).to.be.not.undefined;
            expect(account.data).to.have.property('free');
            expect(account).to.have.property('nonce');
        });
    });
    describe('api system', () => {
        it('call api.rpc.system.chain()', async function () {
            const chain = await api.rpc.system.chain();
            expect(chain).to.contain('Westend')
        });

        it('call api.rpc.system.properties()', async function () {
            const properties = await api.rpc.system.properties();
            expect(properties).to.have.property('ss58Format');
        });

        it('call api.rpc.system.chainType()', async function () {
            const chainType = await api.rpc.system.chainType();
            expect(chainType).to.have.property('isLocal').to.be.true;
        });
    });
    describe('api chain', () => {
        it('call api.rpc.chain.getHeader()', async function () {
            const header = await api.rpc.chain.getHeader();
            expect(header).to.have.property('parentHash').to.have.lengthOf(32);
            expect(header).to.have.property('stateRoot').to.have.lengthOf(32);
            expect(header).to.have.property('extrinsicsRoot').to.have.lengthOf(32);
            expect(header).to.have.property('number');
            expect(header).to.have.property('digest');
        });

        it('call api.rpc.chain.subscribeNewHeads()', async function () {
            let count = 0;
            const unsubHeads = await api.rpc.chain.subscribeNewHeads((lastHeader) => {
                expect(lastHeader).to.have.property('hash').to.have.lengthOf(32);
                expect(lastHeader).to.have.property('number')
                if (++count === 2) {
                    unsubHeads();
                }
            });
        });

        it('call api.rpc.chain.getBlockHash()', async function () {
            const blockHash = await api.rpc.chain.getBlockHash();
            expect(blockHash).to.have.lengthOf(32);
        });

        it('call api.rpc.chain.getBlock()', async function () {
            const block = await api.rpc.chain.getBlock();
            expect(block).to.have.property('block').to.have.property('header');
            const header = block.block.header;
            expect(header).to.have.property('parentHash').to.have.lengthOf(32);
            expect(header).to.have.property('stateRoot').to.have.lengthOf(32);
            expect(header).to.have.property('extrinsicsRoot').to.have.lengthOf(32);
            expect(header).to.have.property('number');
            expect(header).to.have.property('digest');
        });
    });
    describe('api tx', () => {
        it('call api.tx.balances.transfer(ADDR_Bob, 12345).signAndSend(aliceKey)', async function () {
            this.timeout(5000);
            const keyring = new Keyring({type: 'sr25519' });
            const aliceKey = keyring.addFromUri('//Alice');
            const ADDR_Bob = '5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty';

            const transfer = await api.tx.balances.transfer(ADDR_Bob, 12345)
                .signAndSend(aliceKey);

            expect(transfer).to.be.not.null;
            expect(transfer).to.have.lengthOf(32);
        });
    });
    describe('api state', () => {
        it('call api.rpc.state.queryStorage()', async function () {
            const block0Hash = await api.rpc.chain.getBlockHash(0);

            const value = await
                api.rpc.state.queryStorage(["0x1cb6f36e027abb2091cfb5110ab5087f06155b3cd9a8c9e5e9a23fd5dc13a5ed",
                    "0xc2261276cc9d1f8598ea4b6a74b15c2f57c875e4cff74148e4628f264b974c80"], block0Hash);
            expect(value).to.be.not.null;
            const hexString = /^(0x|0X)?[a-fA-F0-9]+$/
            value.forEach(item => {
                expect(item[0]).to.have.lengthOf(32);
                expect(item[0]).to.match( hexString );
                expect(item[1]).to.have.lengthOf(2);
            });
            expect(value[0][0]).to.deep.equal(block0Hash);
        });
    });
   describe('api grandpa', () => {
        it('call api.rpc.grandpa.proveFinality', async function () {
            const proveBlockNumber = 0;
            const finality = await api.rpc.grandpa.proveFinality(proveBlockNumber);
            expect(finality).to.be.not.null;
            expect(finality).to.be.ownProperty('registry')
        });
    });
});
var myRegExp = function (){
    return /.*/;
}