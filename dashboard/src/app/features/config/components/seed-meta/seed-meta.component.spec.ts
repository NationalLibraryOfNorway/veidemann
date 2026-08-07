import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SeedMetaComponent } from './seed-meta.component';

import { of } from 'rxjs';
import { HarnessLoader } from '@angular/cdk/testing';
import { MatFormFieldHarness } from '@angular/material/form-field/testing';
import { TestbedHarnessEnvironment } from '@angular/cdk/testing/testbed';
import { MatInputHarness } from '@angular/material/input/testing';
import { ConfigObject, ConfigRef, Kind, Meta, Seed } from '../../../../shared/models';
import { MatButtonHarness } from '@angular/material/button/testing';
import {MatChipHarness} from '@angular/material/chips/testing';
import { ConfigApiService } from '../../../../core';
import { provideCoreTesting } from '../../../../core/core.testing.module';
import { ActivatedRoute } from '@angular/router';

const exampleMatchingSeeds: ConfigObject[] = [
    {
        kind: Kind.SEED,
        apiVersion: 'v1',
        id: 'existingSeed',
        seed: new Seed({
            disabled: false,
            jobRefList: [],
            entityRef: new ConfigRef({
                kind: Kind.CRAWLENTITY,
                id: 'exampleEntity'
            }),
        }),
        meta: new Meta({
            name: 'http://www.nb.no'
        })
    },
    {
        kind: Kind.SEED,
        apiVersion: 'v1',
        id: 'existingSeed2',
        seed: new Seed({
            disabled: false,
            jobRefList: [],
            entityRef: new ConfigRef({
                kind: Kind.CRAWLENTITY,
                id: 'exampleEntity',
            }),
        }),
        meta: new Meta({
            name: 'https://www.biblotekutvikling.no'
        })
    },
    {
        kind: Kind.SEED,
        apiVersion: 'v1',
        id: 'existingSeed3',
        seed: new Seed({
            disabled: false,
            jobRefList: [],
            entityRef: new ConfigRef({
                kind: Kind.CRAWLENTITY,
                id: 'exampleEntity2'
            }),
        }),
        meta: new Meta({
            name: 'https://www.nb.no'
        })
    },
    {
        kind: Kind.SEED,
        apiVersion: 'v1',
        id: 'existingSeed4',
        seed: new Seed({
            disabled: false,
            jobRefList: [],
            entityRef: new ConfigRef({
                kind: Kind.CRAWLENTITY,
                id: 'exampleEntity3'
            }),
        }),
        meta: new Meta({
            name: 'https://www.bokhylla.no'
        })
    },
];

describe('SeedMetaComponent', () => {
    let component: SeedMetaComponent;
    let fixture: ComponentFixture<SeedMetaComponent>;
    let loader: HarnessLoader;

    let nameFormField: MatFormFieldHarness;
    let nameInput: MatInputHarness;
    const configApiServiceSpy = {
        list: vi.fn().mockName("ConfigApiService.list")
    };

    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [
                SeedMetaComponent,
            ],
            providers: [
                ...provideCoreTesting,
                { provide: ActivatedRoute, useValue: { snapshot: {}, params: of({}), queryParams: of({}) } },
                {
                    provide: ConfigApiService,
                    useValue: configApiServiceSpy,
                },
            ]
        })
            .compileComponents();
    });

    beforeEach(async () => {
        fixture = TestBed.createComponent(SeedMetaComponent);
        loader = TestbedHarnessEnvironment.loader(fixture);
        component = fixture.componentInstance;
        await fixture.whenStable();

        nameFormField = await loader.getHarness<MatFormFieldHarness>(MatFormFieldHarness
            .with({ selector: '[data-testid="name"]' }));
        nameInput = (await nameFormField.getControl()) as MatInputHarness;
    });

    it('should create', () => {
        expect(component).toBeTruthy();
    });

    it('should hide the URL launcher when requested', async () => {
        component.entityRef = new ConfigRef();
        component.updateForm(new Meta());
        await nameInput.setValue('https://www.nb.no');
        fixture.componentRef.setInput('showOpenUrl', false);
        fixture.detectChanges();

        expect(fixture.nativeElement.querySelector('button[mattooltip="Open URL in a new tab"]')).toBeNull();

        fixture.componentRef.setInput('showOpenUrl', true);
        fixture.detectChanges();
        expect(fixture.nativeElement.querySelector('button[mattooltip="Open URL in a new tab"]')).not.toBeNull();
    });
    describe('Creating new seed', async () => {
        it('should validate URL', async () => {
            configApiServiceSpy.list.mockReturnValue([]);
            component.entityRef = new ConfigRef();
            component.updateForm(new Meta());

            /** Valid URLs */

            await nameInput.setValue('http://www.nb.no');
            expect(await nameFormField.isControlValid()).toBeTruthy();

            /** Invalid URLs */

            await nameInput.setValue('');
            await nameInput.blur();
            expect(await nameFormField.isControlValid()).toBeFalsy();

            await nameInput.setValue('http:// shouldfail.com');
            await nameInput.blur();
            expect(await nameFormField.isControlValid()).toBeFalsy();
            expect(await nameFormField.getTextErrors()).toContain('Contains invalid URL(s): http://');
        });

        it('should warn that similar seeds already exists', async () => {
            configApiServiceSpy.list.mockReturnValue(exampleMatchingSeeds);
            component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
            component.updateForm(new Meta());

            await nameInput.setValue('nb.no');
            await nameInput.blur();

            expect(await nameFormField.hasErrors()).toBeTruthy();
            expect(await nameFormField.isControlValid()).toBeFalsy();
            const textErrors = await nameFormField.getTextErrors();
            expect(textErrors).toContain('URL with the same domain already exists for this entity');
            expect(textErrors).toContain('URL with the same domain already exists for another entity');
        });

        describe('Creating multiple new seeds', async () => {

            it('should validate URLs', async () => {
                configApiServiceSpy.list.mockReturnValue([]);
                component.entityRef = new ConfigRef();
                component.updateForm(new Meta());

                /** Valid URLs */

                await nameInput.setValue('http://www.nb.no');
                expect(await nameFormField.isControlValid()).toBeTruthy();
                await nameInput.setValue('http://www.nb.no https://sporbiblioteket.nb.no ' +
                    'https://bibsys-almaprimo.hosted.exlibrisgroup.com/primo-explore/search?vid=NB&lang=no_NO');
                expect(await nameFormField.isControlValid()).toBeTruthy();

                /** Invalid URLs */

                await nameInput.setValue('');
                await nameInput.blur();
                expect(await nameFormField.isControlValid()).toBeFalsy();

                await nameInput.setValue('http:// shouldfail.com http://nb.no http://3628126748');
                await nameInput.blur();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                expect(await nameFormField.getTextErrors()).toContain('Contains invalid URL(s): http://, http://3628126748');
            });

            it('should warn that similar seeds already exists', async () => {
                configApiServiceSpy.list.mockReturnValue(exampleMatchingSeeds);
                component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
                component.updateForm(new Meta());

                await nameInput.setValue('nb.no biblotekutvikling.no');
                await nameInput.blur();

                expect(await nameFormField.hasErrors()).toBeTruthy();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                const textErrors = await nameFormField.getTextErrors();
                expect(textErrors).toContain('URL with the same domain already exists for this entity');
                expect(textErrors).toContain('URL with the same domain already exists for another entity');
            });

            it('should move existing seeds to current entity by clicking button', async () => {
                configApiServiceSpy.list.mockReturnValue(of(exampleMatchingSeeds[3]));
                component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
                component.updateForm(new Meta());
                vi.spyOn(component.move, 'emit');

                await nameInput.setValue('bokhylla.no');
                await nameInput.blur();
                expect(await nameFormField.hasErrors()).toBeTruthy();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                const textErrors = await nameFormField.getTextErrors();
                expect(textErrors).toContain('URL with the same domain already exists for another entity');
                expect(textErrors).not.toContain('URL with the same domain already exists for this entity');

                const moveSeedToEntityButton = await loader.getHarness<MatButtonHarness>(MatButtonHarness
                    .with({ selector: '[data-testid="seedExistsListMoveSeedToEntityButton"]' }));

                await moveSeedToEntityButton.click();

                expect(component.move.emit).toHaveBeenCalledWith({
                    seed: exampleMatchingSeeds[3],
                    entityRef: component.entityRef
                });
            });

            it('should remove a single duplicate url from list by clicking button', async () => {
                configApiServiceSpy.list.mockReturnValue(of(exampleMatchingSeeds[0]));
                component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
                component.updateForm(new Meta());
                vi.spyOn(component, 'onRemoveExistingUrl');

                await nameInput.setValue('http://www.nb.no');
                await nameInput.blur();

                expect(await nameFormField.hasErrors()).toBeTruthy();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                const textErrors = await nameFormField.getTextErrors();

                expect(textErrors).not.toContain('URL with the same domain already exists for another entity');
                expect(textErrors).toContain('URL with the same domain already exists for this entity');

                const duplicateChip = await loader.getHarness(MatChipHarness.with({text: /http:\/\/www\.nb\.no/}));
                await (await duplicateChip.getRemoveButton()).click();

                expect(component.onRemoveExistingUrl)
                    .toHaveBeenCalledWith(exampleMatchingSeeds[0]);
                expect(await nameInput.getValue()).toBe('');
            });

            it('should remove all duplicate seed from input by clicking button', async () => {
                configApiServiceSpy.list.mockReturnValue(of(exampleMatchingSeeds[0], exampleMatchingSeeds[1]));
                component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
                component.updateForm(new Meta());
                vi.spyOn(component, 'onRemoveExistingUrls');

                await nameInput.setValue('http://historiewiki.no https://www.biblotekutvikling.no http://www.nb.no');
                await nameInput.blur();

                expect(await nameFormField.hasErrors()).toBeTruthy();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                const textErrors = await nameFormField.getTextErrors();

                expect(textErrors).not.toContain('URL with the same domain already exists for another entity');
                expect(textErrors).toContain('URL with the same domain already exists for this entity');

                const duplicateChips = await loader.getAllHarnesses(MatChipHarness);
                expect(duplicateChips.length).toEqual(2);

                const removeAllDupButton = await loader.getHarness(MatButtonHarness.with({text: /Remove all from list/}));
                await removeAllDupButton.click();
                expect(component.onRemoveExistingUrls)
                    .toHaveBeenCalledWith([exampleMatchingSeeds[0], exampleMatchingSeeds[1]]);
                await fixture.whenStable();
                await nameInput.blur();
                await fixture.whenStable();
                expect(await nameInput.getValue()).toBe('http://historiewiki.no');
            });

            it('should remove a matching domain without removing similar hostnames', async () => {
                component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
                component.updateForm(new Meta());
                await nameInput.setValue('nb.no notnb.no https://www.biblotekutvikling.no/path');

                component.onRemoveExistingUrl(exampleMatchingSeeds[0]);

                expect(await nameInput.getValue()).toBe('notnb.no\nhttps://www.biblotekutvikling.no/path');
            });
        });

        describe('Updating a seed', async () => {

            it('should validate URL when updating existing seed', async () => {
                configApiServiceSpy.list.mockReturnValue(exampleMatchingSeeds);
                component.entityRef = exampleMatchingSeeds[0].seed.entityRef;
                component.updateForm(new Meta({
                    name: 'thto',
                    created: new Date().toISOString(),
                }));
                await nameInput.blur();
                expect(await nameFormField.hasErrors()).toBeTruthy();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                expect(await nameFormField.getTextErrors()).toContain('Contains invalid URL(s): thto');

                await nameInput.setValue('');
                expect(await nameFormField.hasErrors()).toBeTruthy();
                expect(await nameFormField.isControlValid()).toBeFalsy();
                expect(await nameFormField.getTextErrors()).toContain('The field is required');

                /** Since meta.created is set, async validator should not be active */
                await nameInput.setValue('http://www.nb.no');
                expect(await nameFormField.hasErrors()).toBeFalsy();
                expect(await nameFormField.isControlValid()).toBeTruthy();
            });
        });


    });

});
