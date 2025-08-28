import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CollectionPreviewComponent} from './collection-preview.component';
import {CommonsModule} from '../../../../commons';
import {ConfigObject, Kind} from '../../../../shared/models';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CollectionPreviewComponent', () => {
  let component: CollectionPreviewComponent;
  let fixture: ComponentFixture<CollectionPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CommonsModule],
      declarations: [CollectionPreviewComponent],
      providers: [
        provideZonelessChangeDetection()
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CollectionPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.COLLECTION});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

